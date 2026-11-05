#!/usr/bin/env python3
"""Validate raw trials and produce a self-contained report with request statistics."""
import argparse
import base64
from collections import defaultdict
import csv
import datetime as dt
import gzip
import html
import io
import json
import math
from pathlib import Path
import random
import statistics

PERCENTILES = (0.5, 0.75, 0.9, 0.95, 0.99)
EXPECTED = {(c,s,n) for c in ("json","gob") for s in ("write","read","roundtrip") for n in (4,64)}
PHASES = ("warmup","batch","requests","teardown")

def percentile(values, p):
    """Nearest rank, equal weight per observed request; no interpolation."""
    if not values or not 0 < p <= 1:
        raise ValueError("Empty data or invalid percentile")
    return sorted(values)[math.ceil(p*len(values))-1]

def bootstrap_median(values, seed=20261105):
    rng = random.Random(seed)
    medians = [statistics.median(rng.choices(values,k=len(values))) for _ in range(2000)]
    return [percentile(medians,0.025),percentile(medians,0.975)]

def load(directory):
    directory = Path(directory)
    manifest = json.loads((directory/"manifest.json").read_text())
    archive = directory/"raw.jsonl.gz"
    if archive.exists():
        with gzip.open(archive,"rt") as stream:
            trials = [json.loads(line) for line in stream]
    else:
        trials = [json.loads(p.read_text()) for p in sorted(directory.glob("trial-*.json"))]
    validate(manifest,trials)
    return manifest,trials

def validate(m, trials):
    if not m["complete"] or len(trials) != m["expected_trials"]:
        raise ValueError("Incomplete experiment")
    if m["mode"] not in ("full","smoke") or (m["mode"]=="full" and len(trials)<100):
        raise ValueError("A full experiment requires at least 100 trials")
    if len(trials)%2:
        raise ValueError("AB/BA order requires an even trial count")
    ids = {t["trial"] for t in trials}
    if ids != set(range(1,len(trials)+1)):
        raise ValueError("Missing/duplicate trial numbers")
    if len(m["trials"]) != len(trials) or {t["number"] for t in m["trials"]} != ids:
        raise ValueError("Missing container provenance")
    for field in ("image_id","container_id"):
        if len({r[field] for r in m["trials"]}) != len(trials):
            raise ValueError("Trials must use distinct images and containers")
    hosts = [r["host_config"] for r in m["trials"]]
    if any(h != hosts[0] for h in hosts):
        raise ValueError("Unequal container configuration")
    cfg = m["config"]
    expected_host = {"CpusetCpus":str(cfg["cpu"]),"NanoCpus":1000000000,
                     "Memory":536870912,"MemorySwap":536870912,
                     "NetworkMode":"none","ReadonlyRootfs":True,"PidsLimit":64,
                     "CapDrop":["ALL"],"SecurityOpt":["no-new-privileges"]}
    if hosts[0] != expected_host:
        raise ValueError("Unexpected CPU/memory/isolation settings")
    for r in m["trials"]:
        if r["exit_code"] != 0 or r["rootfs"] != m["rootfs"]:
            raise ValueError("Failed container or different image contents")
        if r["environment"] != m["trials"][0]["environment"]:
            raise ValueError("Unequal environment")
    versions = {(t["go"],t["os"],t["arch"]) for t in trials}
    if len(versions) != 1:
        raise ValueError("Unequal runtime versions")
    for t in trials:
        if t["schema"] != 1 or t["gomaxprocs"] != 1 or t["seed"] != cfg["seed"]:
            raise ValueError("Wrong runtime settings")
        if len(t["results"]) != 12 or {(r["codec"],r["scenario"],r["size"]) for r in t["results"]} != EXPECTED:
            raise ValueError("Missing/duplicate benchmark case")
        orders = {r["order"] for r in t["results"]}
        if orders != set(range(12)):
            raise ValueError("Invalid execution order")
        pairs = defaultdict(dict)
        for r in t["results"]:
            pairs[(r["scenario"],r["size"])][r["codec"]] = r["order"]
            for phase in PHASES:
                want = cfg["samples"] if phase=="requests" else cfg["iterations"]
                if r[phase]["operations"] != want or r[phase]["elapsed_ns"] <= 0:
                    raise ValueError("Invalid phase accounting")
            if len(r["samples"]) != cfg["samples"]:
                raise ValueError("Missing per-request measurements")
            if not math.isclose(r["batch_ns_per_op"],r["batch"]["elapsed_ns"]/cfg["iterations"]):
                raise ValueError("Invalid batch average")
            if r["total_ns"] < sum(r[p]["elapsed_ns"] for p in PHASES):
                raise ValueError("Invalid case duration")
            previous = -1
            for s in r["samples"]:
                if not 0 < s["latency_ns"] or s["at_ns"] < previous or s["at_ns"] > t["elapsed_ns"]:
                    raise ValueError("Invalid timing sample")
                previous = s["at_ns"]
        for order in pairs.values():
            if abs(order["json"]-order["gob"]) != 1 or ((order["json"]<order["gob"]) != (t["trial"]%2==1)):
                raise ValueError("Incorrect paired AB/BA execution")

def summarize(m,trials):
    grouped = defaultdict(list)
    for t in trials:
        for r in t["results"]:
            grouped[(r["size"],r["scenario"],r["codec"])].append((t,r))
    rows = []
    for (size,scenario,codec),values in sorted(grouped.items()):
        requests = [s["latency_ns"] for _,r in values for s in r["samples"]]
        batches = [r["batch_ns_per_op"] for _,r in values]
        row = {"size":size,"scenario":scenario,"codec":codec,"trials":len(values),
               "requests":len(requests),"bytes":values[0][1]["bytes"],
               "batch_median_ns":statistics.median(batches),
               "batch_median_ci95_ns":bootstrap_median(batches),
               "request_mean_ns":statistics.mean(requests),
               "request_stddev_ns":statistics.stdev(requests),
               "request_min_ns":min(requests),"request_max_ns":max(requests),
               "outside_100ns_50ms":sum(x<100 or x>50000000 for x in requests),
               "percentiles_ns":{f"p{int(p*100)}":percentile(requests,p) for p in PERCENTILES},
               "phases":{p:{"operations":sum(r[p]["operations"] for _,r in values),
                           "elapsed_ns":sum(r[p]["elapsed_ns"] for _,r in values)} for p in PHASES},
               "total_ns":sum(r["total_ns"] for _,r in values)}
        rows.append(row)
    comparisons = []
    for size in (4,64):
        for scenario in ("write","read","roundtrip"):
            a = {t["trial"]:r["batch_ns_per_op"] for t,r in grouped[(size,scenario,"json")]}
            b = {t["trial"]:r["batch_ns_per_op"] for t,r in grouped[(size,scenario,"gob")]}
            ratios = [b[i]/a[i] for i in sorted(a)]
            comparisons.append({"size":size,"scenario":scenario,"gob_over_json":statistics.median(ratios),
                                "ci95":bootstrap_median(ratios)})
    overheads = [t["overhead"]["batch_ns_per_op"] for t in trials]
    return {"run_id":m["run_id"],"mode":m["mode"],"trials":len(trials),"rows":rows,
            "comparisons":comparisons,"wall_seconds":m["wall_seconds"],
            "benchmark_seconds":sum(t["elapsed_ns"] for t in trials)/1e9,
            "overhead":{"batch_median_ns":statistics.median(overheads),
                        "batch_p99_ns":percentile(overheads,.99),
                        "trials_above_5ns":sum(x>5 for x in overheads),
                        "timer_mean_ns":statistics.mean(t["overhead"]["timer_mean_ns"] for t in trials),
                        "timer_loop_median_ns":statistics.median(t["overhead"]["timer_loop_ns_per_op"] for t in trials),
                        "timer_min_positive_ns":min(t["overhead"]["timer_min_positive_ns"] for t in trials),
                        "timer_zero_samples":sum(t["overhead"]["timer_zero_samples"] for t in trials),
                        "timer_median_ns":statistics.median(t["overhead"]["timer_p50_ns"] for t in trials),
                        "timer_p99_median_ns":statistics.median(t["overhead"]["timer_p99_ns"] for t in trials)},
            "operations":{p:sum(row["phases"][p]["operations"] for row in rows) for p in PHASES},
            "control_operations":sum(t["overhead"]["operations"] for t in trials),
            "validation_codec_calls":24*len(trials)},grouped

def render(m,trials,output):
    import matplotlib
    matplotlib.use("Agg")
    import matplotlib.pyplot as plt
    import numpy as np
    summary,grouped = summarize(m,trials)
    (output/"summary.json").write_text(json.dumps(summary,ensure_ascii=False,indent=2)+"\n")
    with (output/"statistics.csv").open("w",newline="") as f:
        writer = csv.writer(f, lineterminator="\n")
        writer.writerow(["size","scenario","codec","trials","requests","bytes","batch_median_ns",
                         "p50_ns","p75_ns","p90_ns","p95_ns","p99_ns","min_ns","max_ns"])
        for r in summary["rows"]:
            writer.writerow([r[k] for k in ("size","scenario","codec","trials","requests","bytes","batch_median_ns")]+
                            list(r["percentiles_ns"].values())+[r["request_min_ns"],r["request_max_ns"]])
    parts = ["""<!doctype html><html lang="ru"><meta charset="utf-8"><title>ЛР3 — микробенчмаркинг</title>
<style>body{font:16px/1.55 system-ui,sans-serif;color:#17243b;background:#f4f7fb;max-width:1250px;margin:auto;padding:32px}
section{background:white;border:1px solid #dce3ed;padding:24px;margin:24px 0;border-radius:12px}
h1,h2{line-height:1.2}table{border-collapse:collapse;width:100%;font-variant-numeric:tabular-nums;font-size:14px}
td,th{padding:9px;border-bottom:1px solid #dde4ed;text-align:right}th:first-child,td:first-child{text-align:left}
img{width:100%}.scroll{overflow-x:auto}.note{color:#52627a}code{overflow-wrap:anywhere}</style>
<h1>Лабораторная работа №3</h1><p>JSON и gob: микробенчмарки независимых сообщений в Go.</p>"""]
    parts.append(f"<p>Прогон <code>{html.escape(m['run_id'])}</code> · {len(trials)} испытаний · "
                 f"{html.escape(trials[0]['go'])} · CPU {m['config']['cpu']} · GOMAXPROCS=1</p>")
    if m["mode"] != "full":
        parts.append("<h2>ПРОБНЫЙ ЗАПУСК: не заменяет 100 испытаний</h2>")
    parts.append(f"<section><h2>Условия и время</h2><p>Для каждого испытания создан отдельный образ и свежий контейнер. "
        f"RootFS, бинарный файл и конфигурация одинаковые; различается только метка образа. Порядок JSON/gob — AB/BA. "
        f"Полное время с Docker и очисткой: {summary['wall_seconds']:.2f} с. "
        f"Сумма времени внутри бенчмарков: {summary['benchmark_seconds']:.2f} с.</p>"
        f"<p>SHA-256 бинарного файла: <code>{m['binary_sha256']}</code>.</p>"
        f"<p>Операции: warmup {summary['operations']['warmup']:,}; пакетное измерение {summary['operations']['batch']:,}; "
        f"измерение запросов {summary['operations']['requests']:,}; teardown {summary['operations']['teardown']:,}. "
        f"Дополнительно: {summary['control_operations']:,} пустых контрольных вызовов и "
        f"{summary['validation_codec_calls']:,} проверочных вызовов кодеков вне измерений.</p></section>")
    o = summary["overhead"]
    parts.append(f"<section><h2>Накладные расходы</h2><p>Медиана пустого пакетного запуска: <b>{o['batch_median_ns']:.2f} нс/оп</b>; "
        f"p99 по испытаниям: {o['batch_p99_ns']:.2f} нс/оп. Выше желательных 5 нс: {o['trials_above_5ns']} испытаний. "
        f"Медиана накладных расходов индивидуального таймера: <b>{o['timer_median_ns']:.1f} нс</b>; "
        f"медиана его p99: {o['timer_p99_median_ns']:.1f} нс.</p>"
        f"<p>Средний пустой интервал: {o['timer_mean_ns']:.2f} нс; полный цикл индивидуального таймера "
        f"с сохранением результата: {o['timer_loop_median_ns']:.2f} нс/оп. Минимальный наблюдаемый положительный "
        f"интервал: {o['timer_min_positive_ns']} нс; нулевых контрольных интервалов: {o['timer_zero_samples']:,}. "
        "Нулевая медиана означает квантование часов, а не бесплатный таймер. Точность индивидуальных задержек "
        "ограничена наблюдаемым шагом часов; пакетное усреднение позволяет оценивать меньшие величины.</p>"
        "<p>Пакетная оценка включает цикл, косвенный вызов и два чтения часов на пакет. "
        "Индивидуальный таймер дороже: его стоимость не вычитается из задержек. "
        "Перцентили ниже рассчитаны по отдельным запросам, а не по средним пакетам. "
        "Warmup и teardown в распределения не входят. Выбросы не удаляются.</p></section>")
    origin = min(dt.datetime.fromisoformat(t["started"].replace("Z","+00:00")).timestamp() for t in trials)
    for size in (4,64):
        for scenario in ("write","read","roundtrip"):
            label = {"write":"Запись","read":"Чтение","roundtrip":"Запись + чтение"}[scenario]
            fig,axes = plt.subplots(1,3,figsize=(15,4.4),layout="constrained")
            all_lat = [s["latency_ns"]/1000 for c in ("json","gob") for _,r in grouped[(size,scenario,c)] for s in r["samples"]]
            bins = np.geomspace(min(all_lat),max(all_lat)*1.001,45)
            for codec,color in (("json","#007e87"),("gob","#c64f25")):
                data = grouped[(size,scenario,codec)]
                lat,times = [],[]
                for t,r in data:
                    start = dt.datetime.fromisoformat(t["started"].replace("Z","+00:00")).timestamp()-origin
                    for s in r["samples"]:
                        times.append(start+s["at_ns"]/1e9)
                        lat.append(s["latency_ns"]/1000)
                axes[0].scatter(times,lat,s=2,alpha=.3,label=codec,color=color,rasterized=True)
                q = np.linspace(.001,1,1000)
                sorted_lat = np.sort(lat)
                axes[1].plot(q*100,sorted_lat[np.ceil(q*len(lat)).astype(int)-1],label=f"{codec}: N={len(lat)}",color=color)
                axes[1].scatter([p*100 for p in PERCENTILES],[percentile(lat,p) for p in PERCENTILES],s=20,color=color)
                axes[2].hist(lat,bins=bins,histtype="step",linewidth=1.6,label=codec,color=color)
            axes[0].set(title="Задержка во времени",xlabel="С начала эксперимента, с",ylabel="Задержка, мкс",yscale="log")
            axes[1].set(title="Перцентили запросов",xlabel="Доля запросов, %",ylabel="Задержка, мкс",yscale="log")
            axes[2].set(title="Гистограмма всех запросов",xlabel="Задержка, мкс",ylabel="Количество запросов",xscale="log")
            for ax in axes: ax.grid(alpha=.2); ax.legend(fontsize=8)
            fig.suptitle(f"{label} · записей в сообщении: {size}",fontsize=14)
            name = f"{scenario}-{size}.png"
            buffer = io.BytesIO()
            fig.savefig(buffer,format="png",dpi=130,bbox_inches="tight")
            plt.close(fig)
            (output/name).write_bytes(buffer.getvalue())
            parts.append(f'<section><h2>{label}: число записей — {size}</h2><img alt="Время, перцентили и гистограмма" src="data:image/png;base64,{base64.b64encode(buffer.getvalue()).decode()}">')
            rows = [r for r in summary["rows"] if r["size"]==size and r["scenario"]==scenario]
            parts.append('<div class="scroll"><table><tr><th>Кодек</th><th>Размер, Б</th><th>Пакет, мкс/оп</th><th>p50</th><th>p75</th><th>p90</th><th>p95</th><th>p99</th><th>max</th></tr>')
            for r in rows:
                parts.append(f"<tr><td>{r['codec']}</td><td>{r['bytes']}</td><td>{r['batch_median_ns']/1000:.3f}</td>"+
                             "".join(f"<td>{x/1000:.3f}</td>" for x in r["percentiles_ns"].values())+
                             f"<td>{r['request_max_ns']/1000:.3f}</td></tr>")
            parts.append("</table></div><p class='note'>Задержки запросов — в мкс. Пакет — медиана среднего нс/оп по испытаниям.</p>")
            comp = next(c for c in summary["comparisons"] if c["size"]==size and c["scenario"]==scenario)
            parts.append(f"<p>Медиана парных отношений gob/JSON: {comp['gob_over_json']:.3f}; "
                         f"95% bootstrap-интервал по испытаниям: [{comp['ci95'][0]:.3f}; {comp['ci95'][1]:.3f}]. "
                         "Значение выше 1 означает большее время gob.</p>")
            parts.append('<div class="scroll"><table><tr><th>Кодек</th><th>Warmup, оп / с</th><th>Пакет, оп / с</th><th>Запросы, оп / с</th><th>Teardown, оп / с</th><th>Всего, с</th></tr>')
            for r in rows:
                parts.append(f"<tr><td>{r['codec']}</td>"+"".join(f"<td>{r['phases'][p]['operations']:,} / {r['phases'][p]['elapsed_ns']/1e9:.3f}</td>" for p in PHASES)+
                             f"<td>{r['total_ns']/1e9:.3f}</td></tr>")
            outside = sum(r["outside_100ns_50ms"] for r in rows)
            parts.append(f"</table></div><p>Запросов вне рекомендованных 100 нс–50 мс: {outside}.</p></section>")
    parts.append("<section><h2>Границы интерпретации</h2><p>Каждое сообщение независимо: gob каждый раз передаёт описание типов. "
        "Результаты не описывают долгоживущий gob-поток, сеть, БД или производительность клиники в целом. "
        "Roundtrip — одна логическая операция из последовательных записи и чтения; это не многопоточная нагрузка. "
        "GC включён. CPU закреплён, но Docker Desktop/WSL и процессы хоста могут создавать шум; ядро не эксклюзивное.</p>"
        "<p>95% интервалы получены 2000 bootstrap-перевыборками независимых испытаний (фиксированное зерно). "
        "Перцентили — nearest rank. Соседние запросы одного испытания не считаются независимыми испытаниями. "
        "Число warmup-операций фиксировано; сходимость на любой машине не гарантируется.</p>"
        "<p>Источники: <a href='https://github.com/google/benchmark/blob/main/docs/user_guide.md'>Google Benchmark</a>, "
        "<a href='https://pkg.go.dev/encoding/json'>encoding/json</a>, "
        "<a href='https://pkg.go.dev/encoding/gob'>encoding/gob</a>.</p></section></html>")
    (output/"report.html").write_text("".join(parts),encoding="utf-8")
    print(json.dumps({"trials":len(trials),"overhead":summary["overhead"],"operations":summary["operations"]},indent=2))

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("directory",type=Path)
    args = parser.parse_args()
    m,trials = load(args.directory)
    render(m,trials,args.directory)
