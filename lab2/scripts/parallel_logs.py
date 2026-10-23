import argparse
import os
from pathlib import Path
import re


def latest_logs(directory):
    groups = {}
    for path in directory.glob("parallel-*.log"):
        match = re.fullmatch(r"(parallel-\d{8}-\d{6}-\d+)-\d+\.log", path.name)
        if match:
            groups.setdefault(match[1], []).append(path)
    if not groups:
        raise SystemExit("Нет логов. Сначала выполните make parallel")
    prefix = max(groups, key=lambda value: max(path.stat().st_mtime_ns for path in groups[value]))
    return sorted(groups[prefix], key=lambda path: int(path.stem.rsplit("-", 1)[1]))


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--list", action="store_true")
    args = parser.parse_args()
    logs = latest_logs(Path(__file__).resolve().parents[1] / "reports")
    if args.list:
        print("\n".join(str(path) for path in logs))
    else:
        os.execvp("tail", ["tail", "-n", "20", "-F", *map(str, logs)])
