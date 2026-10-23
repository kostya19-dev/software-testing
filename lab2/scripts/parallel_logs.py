import argparse
from pathlib import Path
import re
import subprocess


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
        names = {str(path): path.stem for path in logs}
        run_id = logs[0].stem
        process = subprocess.Popen(["tail", "-n", "20", "-F", *map(str, logs)], stdout=subprocess.PIPE, text=True)
        try:
            for line in process.stdout:
                header = re.fullmatch(r"==> (.+) <==\n?", line)
                if header:
                    run_id = names.get(header[1], run_id)
                elif line.strip():
                    developer = "developer-" + run_id.rsplit("-", 1)[1]
                    print(f"[{developer}][{run_id}] {line.rstrip()}", flush=True)
        except KeyboardInterrupt:
            pass
        finally:
            process.terminate()
            process.wait()
