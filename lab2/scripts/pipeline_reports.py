import fcntl
import html
import json
import re
import xml.etree.ElementTree as ET


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


def write_summary(directory, state):
    counts = dict(passed=0, failed=0, skipped=0)
    for path in directory.rglob("*junit.xml"):
        try:
            tree = ET.parse(path)
        except ET.ParseError:
            continue
        for case in tree.iter("testcase"):
            status = "failed" if any(case.find(tag) is not None for tag in ("failure", "error")) else "skipped" if case.find("skipped") is not None else "passed"
            counts[status] += 1
    summary = directory / "summary.tmp"
    summary.write_text(json.dumps(dict(state, counts=counts), indent=2, ensure_ascii=False))
    summary.replace(directory / "summary.json")


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
