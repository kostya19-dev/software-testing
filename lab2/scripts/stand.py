import argparse
import fcntl
import hashlib
import json
import os
import subprocess
from pathlib import Path

LAB = Path(__file__).resolve().parents[1]
ENV = dict(os.environ, LAB_ROOT=str(LAB.parent))
SLOTS = 5


def slot_database(slot):
    return "lab2_" + hashlib.sha256(f"stand-slot-{slot}".encode()).hexdigest()[:24]


ENV.update({f"SLOT_DATABASE_{slot}": slot_database(slot) for slot in range(1, SLOTS + 1)})


def snapshot():
    ids = subprocess.check_output(["docker", "ps", "-aq", "--filter", "label=com.docker.compose.project=tiopo-lab2"], text=True).split()
    if not ids:
        return []
    containers = json.loads(subprocess.check_output(["docker", "inspect", *ids], text=True))
    return sorted([{"name": c["Name"], "id": c["Id"], "started": c["State"]["StartedAt"], "restarts": c["RestartCount"]} for c in containers], key=lambda c: c["name"])


def check():
    services = ["mongo", "api", "reports", *[f"api{i}" for i in range(1, SLOTS + 1)]]
    names = [f"tiopo-lab2-{service}-1" for service in services]
    result = subprocess.run(["docker", "inspect", *names], capture_output=True, text=True)
    for container in json.loads(result.stdout or "[]"):
        state = container["State"]
        if not state["Running"] or state.get("Health", {}).get("Status", "healthy") != "healthy":
            raise RuntimeError(f"Стенд недоступен: {container['Name']}")
    if result.returncode:
        raise RuntimeError("Стенд не подготовлен")
    compose("exec", "-T", "--interactive=false", "mongo", "mongosh", "--quiet", "--eval", "quit(db.adminCommand({ping:1}).ok ? 0 : 1)", capture=True)


def lease(run_id, release=False):
    directory = LAB / ".runs"
    directory.mkdir(exist_ok=True)
    with (directory / "slots.lock").open("w") as lock:
        fcntl.flock(lock, fcntl.LOCK_EX)
        path = directory / "slots.json"
        owners = json.loads(path.read_text()) if path.exists() else {}
        owners = {slot: owner for slot, owner in owners.items() if owner != run_id}
        slot = None
        if not release:
            slot = next((i for i in range(1, SLOTS + 1) if str(i) not in owners), None)
            if slot is None:
                raise RuntimeError("Все пять API заняты. Дождитесь завершения сессии или выполните make recover для своей сессии.")
            owners[str(slot)] = run_id
        pending = path.with_suffix(".tmp")
        pending.write_text(json.dumps(owners))
        pending.replace(path)
        return slot


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
    parser.add_argument("action", choices=["up", "down", "status", "check", "snapshot"])
    args = parser.parse_args()
    if args.action == "up":
        up()
        print("Стенд: http://localhost:8094/health")
        print("Отчёты: http://localhost:8093")
    elif args.action == "down":
        slots = LAB / ".runs/slots.json"
        if slots.exists() and json.loads(slots.read_text()):
            raise SystemExit("Есть занятые API. Сначала завершите сессии или выполните make recover RUN_ID=...")
        active = subprocess.run(
            ["docker", "ps", "-q", "--filter", "label=lab2.session=true"],
            check=True, capture_output=True, text=True,
        ).stdout.strip()
        if active:
            raise SystemExit("Есть активные тестовые сессии. Сначала завершите их.")
        compose("down")
    elif args.action == "check":
        check()
        print("Готовый стенд доступен; сервисы не запускались.")
    elif args.action == "snapshot":
        print(json.dumps(snapshot()))
    else:
        compose("ps")


if __name__ == "__main__":
    main()
