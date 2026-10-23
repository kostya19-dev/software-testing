import argparse
import fcntl
import os
import subprocess
from pathlib import Path

LAB = Path(__file__).resolve().parents[1]
ENV = dict(os.environ, LAB_ROOT=str(LAB.parent))


def compose(*args, capture=False):
    return subprocess.run(
        ["docker", "compose", "-p", "tiopo-lab2", "-f", str(LAB / "stand.yaml"), *args],
        env=ENV, check=True, text=True,
        stdout=subprocess.PIPE if capture else None,
    ).stdout


def up():
    (LAB / ".runs").mkdir(exist_ok=True)
    (LAB / "reports").mkdir(exist_ok=True)
    with (LAB / ".runs/stand.lock").open("w") as lock:
        fcntl.flock(lock, fcntl.LOCK_EX)
        compose("up", "-d", "--wait", "--wait-timeout", "360")


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("action", choices=["up", "down", "status"])
    args = parser.parse_args()
    if args.action == "up":
        up()
        print("Стенд: http://localhost:8094/health")
        print("Отчёты: http://localhost:8093")
    elif args.action == "down":
        active = subprocess.run(
            ["docker", "ps", "-q", "--filter", "label=lab2.session=true"],
            check=True, capture_output=True, text=True,
        ).stdout.strip()
        if active:
            raise SystemExit("Есть активные тестовые сессии. Сначала завершите их.")
        compose("down")
    else:
        compose("ps")


if __name__ == "__main__":
    main()
