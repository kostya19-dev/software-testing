import argparse
from datetime import datetime
import hashlib
import json
import subprocess
import tempfile
import uuid
import xml.etree.ElementTree as ET
from pathlib import Path

TITLES = {
    "TestPatientRepositoryLifecycle": "Пациент: создание, чтение, изменение и удаление",
    "TestAppointmentBookingAndCancellation": "Запись: занятое время, отмена и повторная запись",
    "TestScheduleFreeSlotsFilter": "Расписание: фильтрация свободных слотов",
    "TestVisitCompletionAndPatientHistory": "Визит: завершение и история пациента",
    "TestPatientJourney": "Пациент: регистрация → запись → приём → история",
}
STAGES = {"unit": "1. Модульные тесты", "integration": "2. Интеграционные тесты", "e2e": "3. Сквозные тесты"}


def trend_history(directory, state, history_root):
    """Read finished older runs without modifying other testers' reports."""
    current = (state["started"], state["id"])
    previous = []
    for path in history_root.glob("*/summary.json"):
        try:
            other = json.loads(path.read_text())
        except (OSError, json.JSONDecodeError):
            continue
        if (other["started"], other["id"]) < current:
            previous.append((other, path.parent))
    previous.sort(key=lambda item: (item[0]["started"], item[0]["id"]))
    history = []
    for order, (other, folder) in enumerate(previous, 1):
        if not other.get("finished") or other["finished"] > state["started"]:
            continue
        counts = dict(passed=0, failed=0, broken=0, skipped=0, unknown=0)
        for path in folder.rglob("*junit.xml"):
            tree = ET.parse(path)
            for case in tree.iter("testcase"):
                status = "passed"
                for tag, value in (("skipped", "skipped"), ("failure", "failed"), ("error", "broken")):
                    if case.find(tag) is not None:
                        status = value
                counts[status] += 1
        counts["total"] = sum(counts.values())
        history.append({
            "buildOrder": order,
            "reportName": "ЛР2 — " + other["id"],
            "reportUrl": "http://localhost:8093/" + other["id"] + "/index.html",
            "data": counts,
        })
    return len(previous) + 1, list(reversed(history[-19:]))


def generate(directory, history_root=None):
    state = json.loads((directory / "summary.json").read_text())
    build_order, history = trend_history(directory, state, history_root or directory.parent)
    times = {}
    for stage in STAGES:
        events = directory / stage / "events.jsonl"
        if not events.exists():
            continue
        for line in events.read_text().splitlines():
            try:
                event = json.loads(line)
            except json.JSONDecodeError:
                continue
            if not event.get("Test") or event["Action"] not in ("run", "pass", "fail", "skip"):
                continue
            key = (stage, event["Package"], event["Test"])
            field = "start" if event["Action"] == "run" else "stop"
            times.setdefault(key, {})[field] = int(datetime.fromisoformat(event["Time"].replace("Z", "+00:00")).timestamp() * 1000)
    with tempfile.TemporaryDirectory(prefix="allure-lab2-") as temporary:
        work = Path(temporary)
        results = work / "results"
        results.mkdir()
        (results / "history").mkdir()
        (results / "history/history-trend.json").write_text(json.dumps(history, ensure_ascii=False))
        for path in sorted(directory.rglob("*junit.xml")):
            stage = path.parent.name
            try:
                tree = ET.parse(path)
            except ET.ParseError:
                continue
            for case in tree.iter("testcase"):
                name = case.get("name", "")
                identity = stage + "/" + case.get("classname", "") + "/" + name
                result = {
                    "uuid": str(uuid.uuid4()),
                    "historyId": hashlib.sha256(identity.encode()).hexdigest(),
                    "fullName": identity,
                    "name": TITLES.get(name, name),
                    "status": "passed",
                    "stage": "finished",
                    "labels": [
                        {"name": "parentSuite", "value": "ЛР2"},
                        {"name": "suite", "value": STAGES.get(stage, "Стенд и конвейер")},
                        {"name": "package", "value": case.get("classname", "")},
                    ],
                    "parameters": [
                        {"name": "Запуск", "value": state["id"]},
                        {"name": "Разработчик", "value": state.get("developer", "manual")},
                        {"name": "База", "value": state.get("database", "")},
                        {"name": "Go-тест", "value": name},
                    ],
                    "description": "Результат из JUnit. Стенд общий, база и API изолированы по RUN_ID.",
                    "attachments": [],
                    **times.get((stage, case.get("classname", ""), name), {}),
                }
                for tag, status in (("skipped", "skipped"), ("failure", "failed"), ("error", "broken")):
                    failure = case.find(tag)
                    if failure is not None:
                        result["status"] = status
                        result["statusDetails"] = {
                            "message": failure.get("message", ""),
                            "trace": failure.text or "",
                        }
                for tag in ("system-out", "system-err"):
                    if case.findtext(tag):
                        source = str(uuid.uuid4()) + "-attachment.txt"
                        (results / source).write_text(case.findtext(tag))
                        result["attachments"].append({"name": "Вывод теста", "source": source, "type": "text/plain"})
                if stage == "e2e":
                    for filename in ("e2e/events.jsonl", "containers.log", "debug.txt", "rollback.txt", "cleanup.txt", "traffic/requests.txt", "traffic/decoded.txt", "traffic/demo.pcap"):
                        attachment = directory / filename
                        if attachment.is_file():
                            source = hashlib.sha256(filename.encode()).hexdigest() + "-attachment" + attachment.suffix
                            (results / source).write_bytes(attachment.read_bytes())
                            result["attachments"].append({
                                "name": filename, "source": source,
                                "type": "application/vnd.tcpdump.pcap" if attachment.suffix == ".pcap" else "text/plain",
                            })
                (results / (result["uuid"] + "-result.json")).write_text(json.dumps(result, ensure_ascii=False))
        (results / "environment.properties").write_text(
            f'RUN_ID={state["id"]}\nDATABASE={state.get("database", "")}\n'
            f'SOURCE={state.get("commit", "")}\n'
            f'DEVELOPER={state.get("developer", "manual")}\nAPI=api{state.get("slot", "")}\n'
        )
        (results / "executor.json").write_text(json.dumps({
            "name": "Docker", "type": "local", "buildName": state["id"],
            "buildOrder": build_order,
            "reportName": "ЛР2 — " + state["id"],
            "reportUrl": "http://localhost:8093/" + state["id"] + "/index.html",
        }, ensure_ascii=False))
        (results / "categories.json").write_text(json.dumps([
            {"name": "Ошибки тестов и этапов", "matchedStatuses": ["failed"]},
            {"name": "Ошибки выполнения", "matchedStatuses": ["broken"]},
        ], ensure_ascii=False))
        output = work / "html"
        subprocess.run(["allure", "generate", str(results), "--single-file", "-o", str(output)], check=True)
        text = (output / "index.html").read_text()
        text = text.replace('lang="en"', 'lang="ru"', 1)
        text = text.replace("<title>Allure Report</title>", "<title>ЛР2 — Allure</title>", 1)
        refresh = '<meta http-equiv="refresh" content="10">' if not state.get("finished") else ""
        text = text.replace("<head>", """<head>
<style id="lab2-executor-style">
.executor-icon__local { display: none !important; }
</style>
<script>
try {
    const key = "ALLURE_REPORT_SETTINGS";
    const settings = JSON.parse(localStorage.getItem(key) || "{}");
    settings.language = "ru";
    localStorage.setItem(key, JSON.stringify(settings));
} catch (_) {}
</script>""" + refresh, 1)
        pending = directory / "index.tmp"
        pending.write_text(text)
        pending.replace(directory / "index.html")


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("directory", type=Path)
    parser.add_argument("--history-root", type=Path)
    args = parser.parse_args()
    generate(args.directory, args.history_root)
