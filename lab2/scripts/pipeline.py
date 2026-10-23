#!/usr/bin/env python3
import argparse
import datetime as dt
import hashlib
import fcntl
import html
import json
import os
import re
import shutil
import signal
import subprocess as sp
import tempfile
import time
import uuid
import xml.etree.ElementTree as ET
from pathlib import Path

LAB = Path(__file__).resolve().parents[1]
ROOT = LAB.parent
STAGES = ("unit", "integration", "e2e")

import stand


def command(args, *, env=None, cwd=None, capture=False):
    return sp.run(
        [str(x) for x in args],
        cwd=cwd,
        env=env,
        check=True,
        text=True,
        stdout=sp.PIPE if capture else None,
    ).stdout


def junit(path, stage, status, message):
    suite = ET.Element(
        "testsuite",
        name=stage,
        tests="1",
        failures="1" if status == "failed" else "0",
        skipped="1" if status == "skipped" else "0",
        errors="0",
    )
    case = ET.SubElement(suite, "testcase", name="stage_" + stage, classname="pipeline")
    if status != "passed":
        ET.SubElement(
            case, "failure" if status == "failed" else "skipped", message=message
        ).text = message
    ET.indent(suite)
    ET.ElementTree(suite).write(path, encoding="utf-8", xml_declaration=True)


class Run:
    def __init__(self, run_id):
        if not re.fullmatch(r"[a-z0-9][a-z0-9-]{0,63}", run_id):
            raise ValueError(
                "RUN_ID: use up to 64 lowercase letters, digits and hyphens"
            )
        self.id = run_id
        self.project = "tiopo-lab2-" + run_id
        self.run = LAB / ".runs" / run_id
        self.report = LAB / "reports" / run_id
        self.database = "lab2_" + hashlib.sha256(run_id.encode()).hexdigest()[:24]
        self.env = dict(os.environ, RUN_DIR=str(self.run), REPORT_DIR=str(self.report),
                        RUN_ID=run_id, TEST_DATABASE=self.database, LAB_ROOT=str(ROOT))
        self.statefile = self.run / "state.json"
        self.state = (
            json.loads(self.statefile.read_text()) if self.statefile.exists() else {}
        )
        if self.state.get("slot"):
            self.configure_slot(self.state["slot"])

    def configure_slot(self, slot):
        self.database = stand.slot_database(slot)
        self.env.update(TEST_DATABASE=self.database, API_SLOT=str(slot), API_CONTAINER=f"tiopo-lab2-api{slot}-1", DEVELOPER_ID=self.state.get("developer", os.getenv("DEVELOPER_ID", "manual")))

    def save(self):
        pending = self.statefile.with_suffix(".tmp")
        pending.write_text(json.dumps(self.state, indent=2, ensure_ascii=False))
        pending.replace(self.statefile)

    def compose(self, *args, capture=False):
        return command(
            [
                "docker",
                "compose",
                "-p",
                self.project,
                "-f",
                self.run / "compose.yaml",
                *args,
            ],
            env=self.env,
            capture=capture,
        )

    def prepare(self):
        if self.run.exists() or self.report.exists():
            raise ValueError(
                "RUN_ID already exists: use a new ID, or make recover RUN_ID=..."
            )
        self.run.mkdir(parents=True)
        self.report.mkdir(parents=True)
        self.state = {
            "id": self.id,
            "developer": os.getenv("DEVELOPER_ID", "manual"),
            "database": self.database,
            "keep_failed": os.getenv("LAB2_KEEP_FAILED") == "1",
            "started": dt.datetime.now(dt.timezone.utc).isoformat(),
            "stages": {s: "skipped" for s in STAGES},
            "baseline": False,
            "cleaned": False,
        }
        self.save()
        for stage in STAGES:
            (self.report / stage).mkdir()
            junit(
                self.report / stage / "junit.xml",
                stage,
                "skipped",
                "Stage not executed (a prerequisite failed or the run was interrupted)",
            )
        (self.report / "traffic").mkdir()
        (self.run / "baseline").mkdir()
        try:
            stand.check()
            slot = stand.lease(self.id)
            self.configure_slot(slot)
            self.state.update(slot=slot, database=self.database)
            self.save()
            print(f"DEVELOPER_ID={self.state['developer']} RUN_ID={self.id} API=api{slot} DATABASE={self.database}", flush=True)
        except Exception as exc:
            self.state.update(preparation_error=str(exc), cleaned=True, finished=dt.datetime.now(dt.timezone.utc).isoformat())
            self.save()
            junit(self.report / "prepare-junit.xml", "prepare", "failed", str(exc))
            self.render()
            raise
        shutil.copyfile(LAB / "compose.yaml", self.run / "compose.yaml")
        shutil.copyfile(
            ROOT / "project/VitaLineClinic/migrations/mongo/001-init.js",
            self.run / "init.js",
        )
        with tempfile.TemporaryDirectory(prefix="lab2-source-") as temporary:
            checkout = Path(temporary)
            ignore = shutil.ignore_patterns(
                ".git",
                ".env",
                ".env.*",
                ".runs",
                "reports",
                ".tools",
                "node_modules",
                "bin",
                "logs",
                "__pycache__",
                "*.log",
                "*.pdf",
                "*.png",
                "*.exe",
            )
            for source in ("project/VitaLineClinic", "lab1", "lab2"):
                shutil.copytree(ROOT / source, checkout / source, ignore=ignore)
            command(["git", "init", "-q", "-b", "main", checkout])
            command(["git", "add", "."], cwd=checkout)
            command(
                [
                    "git",
                    "-c",
                    "user.name=Lab 2 snapshot",
                    "-c",
                    "user.email=lab2@example.test",
                    "commit",
                    "-qm",
                    "Source snapshot for " + self.id,
                ],
                cwd=checkout,
            )
            self.state["commit"] = command(
                ["git", "rev-parse", "HEAD"], cwd=checkout, capture=True
            ).strip()
            command(
                ["git", "bundle", "create", self.run / "source.bundle", "--all"],
                cwd=checkout,
            )
        self.save()
        (self.report / "source.txt").write_text(self.state["commit"] + "\n")
        for image in ("tiopo-lab2-runner:2", "mongo:7"):
            with (self.report / "images.txt").open("a") as out:
                out.write(
                    command(
                        [
                            "docker",
                            "image",
                            "inspect",
                            "--format",
                            "{{.Id}} {{json .RepoDigests}}",
                            image,
                        ],
                        capture=True,
                    )
                )
        self.render()

    def mongo(self, *args, capture=False):
        return stand.compose("exec", "-T", "--interactive=false", "mongo", *args, capture=capture)

    def fingerprint(self):
        script = 'const d=db.getSiblingDB(' + json.dumps(self.database) + '); print(EJSON.stringify(d.getCollectionNames().sort().map(n=>({name:n,documents:d.getCollection(n).find().sort({_id:1}).toArray(),indexes:d.getCollection(n).getIndexes().sort((a,b)=>a.name.localeCompare(b.name))}))));'
        result = self.mongo("mongosh", "--quiet", "--eval", script, capture=True)
        return hashlib.sha256(result.strip().encode()).hexdigest()

    def baseline(self):
        if self.state.get("slot") is None or self.database != stand.slot_database(self.state["slot"]):
            raise RuntimeError("Нет выделенного API и базы")
        self.state["database_owned"] = True
        self.save()
        self.mongo("mongosh", "--quiet", "--eval", "db.getSiblingDB(" + json.dumps(self.database) + ").dropDatabase()")
        self.mongo(
            "env", "MONGODB_DATABASE=" + self.database, "mongosh", "--quiet", f"/runs/{self.id}/init.js"
        )
        self.mongo(
            "mongodump",
            "--quiet",
            "--db",
            self.database,
            f"--archive=/runs/{self.id}/baseline/initial.archive",
        )
        self.state["fingerprint"] = self.fingerprint()
        self.state["baseline"] = True
        self.save()

    def rollback(self, label):
        if not self.state.get("baseline"):
            return
        self.compose("stop", "-t", "10", "tests", "capture", "debug")
        containers = command(["docker", "ps", "-q", "--filter",
                              "label=com.docker.compose.project=" + self.project], capture=True).split()
        if containers:
            command(["docker", "stop", "-t", "10", *containers])
        self.mongo(
            "mongosh",
            "--quiet",
            "--eval",
            "db.getSiblingDB(" + json.dumps(self.database) + ").dropDatabase()",
        )
        self.mongo("mongorestore", "--quiet", f"--archive=/runs/{self.id}/baseline/initial.archive")
        actual = self.fingerprint()
        okay = actual == self.state["fingerprint"]
        with (self.report / "rollback.txt").open("a") as out:
            out.write(
                f"{label}: {'PASS' if okay else 'FAIL'}\nbefore={self.state['fingerprint']}\nafter ={actual}\n"
            )
        if not okay:
            raise RuntimeError(
                "Database rollback does not match its original documents and indexes"
            )

    def tests(self, stage):
        self.compose("run", "--rm", "--no-deps", "-T", "--interactive=false", "tests", stage)

    def start_capture(self):
        self.compose("up", "-d", "--no-deps", "capture")
        for _ in range(50):
            if "listening on" in self.compose("logs", "--no-color", "capture", capture=True):
                return
            time.sleep(0.2)
        raise RuntimeError("tcpdump did not start")

    def stop_capture(self):
        self.compose("stop", "-t", "10", "capture")
        if not (self.report / "traffic/demo.pcap").exists():
            return
        output = self.compose(
            "run", "--rm", "--no-deps", "-T", "--interactive=false", "--entrypoint", "tcpdump",
            "tests", "-nn", "-A", "-r", "/reports/traffic/demo.pcap", capture=True,
        )
        (self.report / "traffic/decoded.txt").write_text(output)

    def logs(self):
        (self.report / "containers.log").write_text(
            stand.compose("logs", "--no-color", "mongo", "api" + str(self.state["slot"]), capture=True)
            + self.compose("logs", "--no-color", capture=True)
        )

    def debug(self):
        stand.check()
        if not self.state.get("baseline"):
            self.baseline()
        self.compose("up", "-d", "--no-deps", "debug")
        for _ in range(180):
            if "API server listening at:" in self.compose("logs", "--no-color", "debug", capture=True):
                break
            time.sleep(1)
        else:
            raise RuntimeError("Delve не запустился; см. docker compose logs debug")
        self.state["kept"] = True
        self.state["debugging"] = True
        self.save()
        self.render()
        address = self.compose("port", "debug", "40000", capture=True).strip()
        print("Delve: " + address, flush=True)
        print("Исходники: /src/lab2/e2e. Остановка: make recover RUN_ID=" + self.id, flush=True)

    def stage(self, stage):
        for previous in STAGES[: STAGES.index(stage)]:
            if self.state["stages"][previous] != "passed":
                raise RuntimeError(f"{previous} must pass before {stage}")
        self.state["stages"][stage] = "running"
        self.save()
        self.render()
        (self.report / stage / "junit.xml").unlink(missing_ok=True)
        error = None
        try:
            stand.check()
            if stage == "integration":
                self.baseline()
            if stage == "e2e":
                self.start_capture()
            self.tests(stage)
            if os.getenv("LAB2_FAIL_STAGE") == stage:
                if self.state.get("baseline"):
                    self.mongo(
                        "mongosh",
                        "--quiet",
                        "--eval",
                        'db.getSiblingDB(' + json.dumps(self.database) + ').fault_probe.insertOne({_id:"rollback-proof"})',
                    )
                raise RuntimeError(
                    "Intentional CI failure demonstration after real tests; not a product defect"
                )
            if stage == "e2e":
                self.tests("demo")
        except BaseException as exc:
            error = exc
        finally:
            if stage == "e2e":
                try:
                    self.stop_capture()
                    self.logs()
                except BaseException as exc:
                    error = error or exc
            if stage != "unit" and not (error and self.state.get("keep_failed")):
                try:
                    self.rollback(stage)
                except BaseException as exc:
                    error = exc
            self.state["stages"][stage] = "failed" if error else "passed"
            if error:
                junit(
                    self.report / stage / "pipeline-junit.xml",
                    stage,
                    "failed",
                    str(error),
                )
                if not (self.report / stage / "junit.xml").exists():
                    junit(
                        self.report / stage / "junit.xml",
                        stage,
                        "skipped",
                        "Test execution did not produce results; see pipeline error",
                    )
            self.save()
            self.render()
        if error:
            raise error

    def finish(self, force=False):
        if not self.state or self.state.get("cleaned"):
            stand.lease(self.id, release=True)
            return
        for stage, status in self.state["stages"].items():
            if status == "running":
                self.state["stages"][stage] = "failed"
                junit(
                    self.report / stage / "pipeline-junit.xml",
                    stage,
                    "failed",
                    "Run interrupted before the stage completed",
                )
        if not force and self.state.get("keep_failed") and "failed" in self.state["stages"].values():
            self.state["kept"] = True
            self.state["finished"] = dt.datetime.now(dt.timezone.utc).isoformat()
            self.logs()
            self.save()
            self.render()
            print("Сессия сохранена для отладки: " + self.id, flush=True)
            return
        errors = []
        try:
            self.logs()
            self.rollback("finalize")
            if self.state.get("database_owned", self.state.get("baseline")):
                self.mongo("mongosh", "--quiet", "--eval",
                           "db.getSiblingDB(" + json.dumps(self.database) + ").dropDatabase()")
        except BaseException as exc:
            errors.append(str(exc))
        try:
            self.compose("down", "--volumes", "--remove-orphans", "--timeout", "10")
            remaining = command(
                [
                    "docker",
                    "ps",
                    "-aq",
                    "--filter",
                    "label=com.docker.compose.project=" + self.project,
                ],
                capture=True,
            ).strip()
            volumes = command(
                [
                    "docker",
                    "volume",
                    "ls",
                    "-q",
                    "--filter",
                    "label=com.docker.compose.project=" + self.project,
                ],
                capture=True,
            ).strip()
            if remaining or volumes:
                raise RuntimeError("Stand resources remain after cleanup")
            self.state["cleaned"] = not errors
            self.state["kept"] = bool(errors)
            (self.report / "cleanup.txt").write_text(
                ("FAIL: database cleanup incomplete; " if errors else "PASS: ")
                + "session containers and private networks removed for " + self.project
                + "\nThe shared stand remains running.\n"
            )
        except BaseException as exc:
            errors.append(str(exc))
        if errors:
            junit(
                self.report / "cleanup-junit.xml",
                "cleanup",
                "failed",
                "; ".join(errors),
            )
        self.state["finished"] = dt.datetime.now(dt.timezone.utc).isoformat()
        self.save()
        self.render()
        if errors:
            raise RuntimeError("; ".join(errors))
        stand.lease(self.id, release=True)
        self.prune_runtime()

    def prune_runtime(self):
        if not self.state.get("cleaned"):
            return
        for name in ("source.bundle", "baseline", "init.js", "compose.yaml"):
            path = self.run / name
            resolved = path.resolve()
            if not resolved.is_relative_to(self.run.resolve()) or path.is_symlink():
                raise RuntimeError("Unsafe runtime cleanup path: " + str(path))
            if path.is_dir():
                shutil.rmtree(path)
            elif path.exists():
                path.unlink()

    def render(self):
        counts = dict(passed=0, failed=0, skipped=0)
        for path in self.report.rglob("*junit.xml"):
            try:
                tree = ET.parse(path)
            except ET.ParseError:
                continue
            for case in tree.iter("testcase"):
                status = "failed" if any(case.find(tag) is not None for tag in ("failure", "error")) else "skipped" if case.find("skipped") is not None else "passed"
                counts[status] += 1
        summary = self.report / "summary.tmp"
        summary.write_text(json.dumps(dict(self.state, counts=counts), indent=2, ensure_ascii=False))
        summary.replace(self.report / "summary.json")
        command([
            "docker", "run", "--pull=never", "--rm", "--label", "lab2.report=" + self.id, "--network", "none",
            "-v", str(self.report) + ":/reports",
            "-v", str(LAB / "reports") + ":/all-reports:ro",
            "-v", str(LAB / "scripts/report.py") + ":/report.py:ro",
            "--entrypoint", "python3", "tiopo-lab2-runner:2", "/report.py", "/reports",
            "--history-root", "/all-reports",
        ])
        render_index(LAB)
        if os.geteuid() == 0:
            owner = LAB.stat()
            for directory in (self.run, self.report):
                for path in [directory, *directory.rglob("*")]:
                    os.chown(path, owner.st_uid, owner.st_gid)


def interrupted(signum, frame):
    raise KeyboardInterrupt(f"Interrupted by signal {signum}")


def render_index(lab):
    with (lab / ".runs/reports.lock").open("w") as lock:
        fcntl.flock(lock, fcntl.LOCK_EX)
        states = [json.loads(path.read_text()) for path in (lab / "reports").glob("*/summary.json")]
        states.sort(key=lambda data: (data["started"], data["id"]), reverse=True)

        def group(run_id):
            match = re.fullmatch(r"((?:parallel|five)-\d{8}-\d{6}-\d+)-\d+", run_id)
            return match[1] if match else run_id

        latest = group(states[0]["id"]) if states else None
        current, previous = [], []
        for data in states:
            active = not data.get("finished") and "failed" not in data["stages"].values()
            stages = " → ".join("ожидает" if active and value == "skipped" else value for value in data["stages"].values())
            if data.get("preparation_error"):
                stages = "Подготовка: failed — " + data["preparation_error"].split(". Тесты не восстанавливают сервисы.")[0]
            row = f'<li><a href="{html.escape(data["id"])}/index.html">Allure: {html.escape(data["id"])}</a> — {html.escape(stages)}</li>'
            (current if group(data["id"]) == latest else previous).append(row)
        text = '<!doctype html><html lang="ru"><head><meta charset="utf-8"><meta http-equiv="refresh" content="10"><title>ЛР2 — запуски</title></head><body><h1>ЛР2 — отчёты Allure</h1><ul>' + "".join(current) + '</ul>'
        if previous:
            text += f'<details><summary>Предыдущие запуски ({len(previous)})</summary><ul>' + "".join(previous) + '</ul></details>'
        pending = lab / "reports/index.tmp"
        pending.write_text(text + '</body></html>')
        pending.replace(lab / "reports/index.html")


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "action", choices=["all", "prepare", "unit", "integration", "e2e", "finish", "debug", "connect"]
    )
    parser.add_argument(
        "--id",
        default=os.getenv("RUN_ID")
        or dt.datetime.now().strftime("%Y%m%d-%H%M%S-") + uuid.uuid4().hex[:6],
    )
    args = parser.parse_args()
    if not re.fullmatch(r"[a-z0-9][a-z0-9-]{0,63}", args.id):
        parser.error("RUN_ID: строчные латинские буквы, цифры и дефисы, до 64 символов")
    (LAB / ".runs").mkdir(exist_ok=True)
    with (LAB / ".runs" / (args.id + ".lock")).open("w") as lock:
        try:
            fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
        except BlockingIOError:
            raise SystemExit("Этот RUN_ID уже выполняется: " + args.id)
        run = Run(args.id)
        print("Lab 2 run: " + args.id, flush=True)
        signal.signal(signal.SIGTERM, interrupted)
        if args.action == "all":
            try:
                run.prepare()
                for stage in STAGES:
                    run.stage(stage)
            finally:
                run.finish()
        elif args.action == "prepare":
            run.prepare()
        elif args.action == "finish":
            run.finish(force=True)
        elif args.action == "connect":
            run.compose("exec", "debug", "dlv", "connect", "127.0.0.1:40000")
        elif args.action == "debug":
            if not run.state:
                run.prepare()
            elif run.state.get("cleaned"):
                raise SystemExit("Сессия уже очищена. Выберите новый RUN_ID для отладки.")
            run.debug()
        else:
            run.stage(args.action)
        print("Allure: http://localhost:8093/" + args.id + "/index.html", flush=True)


if __name__ == "__main__":
    main()
