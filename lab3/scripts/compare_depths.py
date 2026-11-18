#!/usr/bin/env python3
import argparse
import datetime as dt
import html
import json
import os
from pathlib import Path
import tempfile

import run as runner
from report import load, summarize, read_summary, SCENARIOS

PROFILES = ("baseline", "depth5")


def compare_reports(baseline, nested, output):
    runs = {}
    reports = []
    for name, directory in zip(PROFILES, (baseline, nested)):
        m, trials = load(directory)
        summary, _ = summarize(m, trials)
        if read_summary(directory) != summary:
            raise ValueError(f"{name}: summary differs from measurements")
        depth = 3 if name == "baseline" else 5
        for item in m.get("inputs", []):
            if item["depth"] != depth:
                raise ValueError(f"{name}: unexpected input depth")
        runs[name] = {(r["scenario"], r["size"], r["codec"]):r for r in summary["rows"]}
        reports.append((directory, m))
    first, second = [m for _, m in reports]
    for key in ("config", "expected_trials", "mode"):
        if first[key] != second[key]:
            raise ValueError(f"Unequal experiments: {key}")
    if "inputs" in first and "inputs" in second:
        for size in (4,64):
            patients = []
            for directory, m in reports:
                info = next(i for i in m["inputs"] if i["visits"] == size)
                patients.append(json.loads((directory / Path(info["file"]).name).read_text()))
            if "Clinic" not in patients[1]:
                raise ValueError("Missing nested Clinic")
            del patients[1]["Clinic"]
            if patients[0] != patients[1]:
                raise ValueError(f"Patient fields differ between profiles ({size} visits)")

    import matplotlib
    matplotlib.use("Agg")
    import matplotlib.pyplot as plt
    output.mkdir(parents=True, exist_ok=True)
    parts = ['<!doctype html><html lang="ru"><meta charset="utf-8"><title>Сравнение структур пациента</title>',
             '<style>body{font:16px/1.5 system-ui;max-width:1000px;margin:auto;padding:2em}img{max-width:100%}section{margin:2em 0}</style>',
             "<h1>Исходный пациент и пациент с Clinic</h1>",
             "<p>Показаны p50 и p99 индивидуальных задержек в наносекундах. "
             "При добавлении Clinic меняются и глубина, и размер сообщения. У каждого варианта свой прогон.</p>"]
    if first["mode"] == "smoke":
        parts.append("<p><strong>Пробный запуск</strong></p>")
    for name, (directory, m) in zip(PROFILES, reports):
        url = html.escape(os.path.relpath(directory/"report.html", output), quote=True)
        parts.append(f'<p><a href="{url}">{name}: полный отчёт</a> · {m["expected_trials"]} испытаний · {html.escape(m["run_id"])}</p>')

    for operation in SCENARIOS:
        for size in (4,64):
            fig, axes = plt.subplots(1, 2, figsize=(9,3.7), layout="constrained")
            for ax, metric in zip(axes, ("p50","p99")):
                for i, (codec,color) in enumerate((("json","#007e87"),("gob","#c64f25"))):
                    values = [runs[name][operation,size,codec]["percentiles_ns"][metric] for name in PROFILES]
                    bars = ax.bar([x+(i-0.5)*0.36 for x in (0,1)],values,width=0.34,color=color,label=codec)
                    ax.bar_label(bars,fmt="%.0f",padding=2,fontsize=8)
                ax.set_xticks([0,1],["Исходный","Глубина 5"])
                ax.set_ylabel("Задержка, нс")
                ax.set_title(metric)
                ax.margins(y=0.25)
            axes[1].legend(loc="upper left", bbox_to_anchor=(1.01,1))
            fig.suptitle(f"{operation} · посещений: {size}")
            name = f"compare-{operation}-{size}.png"
            fig.savefig(output/name,dpi=120)
            plt.close(fig)
            parts.append(f'<section><img src="{name}" alt="{operation}, {size} посещений"></section>')
    (output/"index.html").write_text("".join(parts)+"</html>",encoding="utf-8")
    print(f"COMPARISON={output/'index.html'}",flush=True)


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--smoke",action="store_true")
    p.add_argument("--trials",type=int,default=100)
    p.add_argument("--cpu",type=int,default=0)
    p.add_argument("--iterations",type=int,default=256)
    p.add_argument("--samples",type=int,default=256)
    p.add_argument("--allure",action="store_true")
    args=p.parse_args()
    if args.smoke:
        args.trials, args.iterations, args.samples = 2,32,32
    parent=runner.ROOT/"results/comparisons"
    parent.mkdir(parents=True,exist_ok=True)
    stamp=dt.datetime.now(dt.timezone.utc).strftime("%Y%m%d-%H%M%S-")
    args.output_root=Path(tempfile.mkdtemp(dir=parent,prefix=stamp))
    directories=[]
    for args.profile in PROFILES:
        print(f"PROFILE={args.profile}", flush=True)
        directories.append(runner.run(args))
    compare_reports(*directories,args.output_root)


if __name__=="__main__":
    main()
