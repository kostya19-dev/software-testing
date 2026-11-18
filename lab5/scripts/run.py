import datetime
import os
from pathlib import Path
import secrets
import signal
import subprocess
import sys
import uuid

from report import render

ROOT = Path(__file__).resolve().parents[1]


def main():
    def interrupted(signum, frame):
        raise KeyboardInterrupt

    signal.signal(signal.SIGTERM, interrupted)
    env = os.environ.copy()
    if not env.get("LAB5_TEST_PASSWORD"):
        if env.get("CI"):
            print("CI requires the LAB5_TEST_PASSWORD Actions secret", file=sys.stderr)
            return 2
        env["LAB5_TEST_PASSWORD"] = secrets.token_urlsafe(32)
    password = env["LAB5_TEST_PASSWORD"]
    if not 16 <= len(password.encode()) <= 72:
        print("LAB5_TEST_PASSWORD must be 16..72 UTF-8 bytes", file=sys.stderr)
        return 2
    run_id = datetime.datetime.now(datetime.timezone.utc).strftime("%Y%m%d-%H%M%S") + "-" + uuid.uuid4().hex[:10]
    results = ROOT / "results" / run_id
    results.mkdir(parents=True)
    env["LAB5_RESULTS"] = str(results)
    project = "lab5-" + run_id
    compose = ["docker", "compose", "-p", project, "-f", str(ROOT / "compose.yaml")]
    print("Run:", run_id, flush=True)

    def call(args, log=None):
        return subprocess.run(compose + args, cwd=ROOT, env=env,
                              stdout=log, stderr=subprocess.STDOUT if log else None).returncode

    code = 1
    cleanup = 1
    try:
        code = call(["build", "api", "tests"])
        if code == 0:
            code = call(["up", "-d", "--wait", "--wait-timeout", "120", "mongo", "mail", "api"])
        if code == 0:
            with (results / "bdd.log").open("w") as log:
                code = call(["run", "--rm", "--no-deps", "-T", "tests"], log)
            content = (results / "bdd.log").read_text()
            print(content.replace(password, "[REDACTED]"))
    except KeyboardInterrupt:
        print("Run interrupted; removing its stand", file=sys.stderr)
        code = 130
    finally:
        cleanup = call(["down", "--volumes", "--remove-orphans", "--timeout", "10"])
        (results / "cleanup.txt").write_text("isolated stand removed\n" if cleanup == 0 else "cleanup FAILED\n")
    leaked = False
    for path in results.iterdir():
        if path.is_file():
            data = path.read_bytes()
            if password.encode() in data:
                leaked = True
                path.write_bytes(data.replace(password.encode(), b"[REDACTED]"))
    if leaked:
        print("Secret detected in test output; redacted, run failed", file=sys.stderr)
    render(results, 1 if code or leaked else 0, cleanup)
    print("Reports:", results)
    return 1 if code or cleanup or leaked else 0


if __name__ == "__main__":
    sys.exit(main())
