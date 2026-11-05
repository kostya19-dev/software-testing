#!/usr/bin/env python3
"""Export a complete full run; refuses to overwrite existing evidence."""
import argparse
import gzip
import hashlib
import json
from pathlib import Path
import shutil
from report import load, summarize

def archive(source, target):
    m,trials = load(source)
    if m["mode"] != "full":
        raise ValueError("Only a complete full experiment can be published")
    if target.exists() and any(target.iterdir()):
        raise ValueError("Destination is not empty; choose a new directory")
    target.mkdir(parents=True,exist_ok=True)
    s,_ = summarize(m,trials)
    if json.loads((source/"summary.json").read_text()) != s:
        raise ValueError("Report does not match raw measurements")
    for name in ("manifest.json","summary.json","statistics.csv","report.html"):
        shutil.copy2(source/name,target/name)
    for name in [f"{scenario}-{size}.png" for scenario in ("read","write","roundtrip") for size in (4,64)]:
        shutil.copy2(source/name,target/name)
    with gzip.GzipFile(filename=str(target/"raw.jsonl.gz"),mode="wb",mtime=0) as f:
        for t in trials:
            f.write((json.dumps(t,ensure_ascii=False,separators=(",",":"))+"\n").encode())
    lines = ["# Результаты полного эксперимента", "",
        f"Дата UTC: {m['started_utc']}. Идентификатор: `{m['run_id']}`.", "",
        f"**{len(trials)} отдельных образов, {len(trials)} новых контейнеров, 12 комбинаций, "
        f"{sum(s['operations'].values()):,} логических операций.**",
        "На каждую комбинацию приходится 100 испытаний и 25 600 индивидуальных задержек.", "",
        "Откройте [report.html](report.html) локально в браузере. Все рисунки встроены в HTML.", "",
        f"Окружение: {trials[0]['go']}, {trials[0]['os']}/{trials[0]['arch']}, "
        f"Docker {m['docker']['ServerVersion']}, WSL/Docker Desktop, CPU {m['config']['cpu']}.",
        f"Полное время с Docker и очисткой: {s['wall_seconds']:.2f} с; "
        f"сумма времени бенчмарков: {s['benchmark_seconds']:.2f} с.", "",
        "## Сравнение", "",
        "Медиана **парного** отношения пакетного времени gob/JSON; >1 — gob медленнее.", "",
        "| Записей | Сценарий | gob/JSON | 95% bootstrap CI |",
        "|---:|---|---:|---|"]
    for c in s["comparisons"]:
        lines.append(f"| {c['size']} | {c['scenario']} | {c['gob_over_json']:.3f} | "
                     f"[{c['ci95'][0]:.3f}; {c['ci95'][1]:.3f}] |")
    lines += ["", "Результат зависит от размера сообщения и операции. "
              "Универсального победителя из этого сравнения выводить нельзя: "
              "каждый gob-поток здесь создаётся заново и содержит описание типов.",
              "", "## Измерительный инструмент", "",
              f"Медиана пакетного overhead: {s['overhead']['batch_median_ns']:.3f} нс/оп; "
              f"p99 по испытаниям: {s['overhead']['batch_p99_ns']:.3f} нс/оп. "
              f"Испытаний выше желательных 5 нс: {s['overhead']['trials_above_5ns']}.",
              f"Средний пустой индивидуальный интервал: {s['overhead']['timer_mean_ns']:.2f} нс; "
              f"полный цикл таймера и сохранения: {s['overhead']['timer_loop_median_ns']:.2f} нс/оп. "
              f"Минимальный наблюдаемый положительный интервал: {s['overhead']['timer_min_positive_ns']} нс.",
              "Нулевая медиана короткого таймера обусловлена квантованием; это не нулевая стоимость измерения.",
              f"За пределами 100 нс–50 мс: {sum(r['outside_100ns_50ms'] for r in s['rows'])} "
              "индивидуальных задержек. Выбросы не отбрасывались.",
              "", "## Фазы", "",
              "| Фаза | Логических операций |", "|---|---:|"]
    lines += [f"| {p} | {v:,} |" for p,v in s["operations"].items()]
    lines += ["", "Warmup и teardown не участвуют в основных распределениях. "
              "Контрольные пустые вызовы и проверка корректности учтены отдельно в summary.",
              "", "## Проверка", "", "```bash", "cd lab3",
              "python3 scripts/verify.py evidence", "```", "",
              "- `raw.jsonl.gz`: все исходные измерения и времена фаз.",
              "- `manifest.json`: image/container ID, настройки CPU/RAM и SHA-256 бинарника.",
              "- `summary.json`, `statistics.csv`: статистические показатели.",
              "- `SHA256SUMS`: контрольные суммы всех файлов этого каталога.",
              "", "Проверка пересчитывает статистику из raw и сравнивает её с опубликованной.",
              "Графики на GitHub:", ""]
    for size in (4,64):
        for scenario in ("write","read","roundtrip"):
            lines.append(f"![{scenario}, {size} записей]({scenario}-{size}.png)")
            lines.append("")
    (target/"README.md").write_text("\n".join(lines).rstrip()+"\n",encoding="utf-8")
    hashes = [f"{hashlib.sha256(p.read_bytes()).hexdigest()}  {p.name}" for p in sorted(target.iterdir()) if p.is_file()]
    (target/"SHA256SUMS").write_text("\n".join(hashes)+"\n")
    print(target)

if __name__ == "__main__":
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("source",type=Path)
    p.add_argument("target",type=Path)
    a = p.parse_args()
    archive(a.source,a.target)
