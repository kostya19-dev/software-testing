import csv
import io
import json
from pathlib import Path
import shutil
import sys
import xml.etree.ElementTree as ET

SOURCE = {"type": "grafana-testdata-datasource", "uid": "godog-junit"}


def query(columns, rows):
    stream = io.StringIO()
    writer = csv.writer(stream, lineterminator="\n")
    writer.writerow(columns)
    writer.writerows(rows)
    return {"refId": "A", "datasource": SOURCE, "scenarioId": "csv_content", "csvContent": stream.getvalue()}


def scenario_status(case):
    if case.find("failure") is not None:
        return "Ошибка"
    if case.get("status") == "skipped" or case.find("skipped") is not None:
        return "Пропущен"
    if case.get("status", "passed") == "passed" and case.find("error") is None:
        return "Пройден"
    return "Ошибка"


def build(report, output):
    root = ET.parse(report).getroot()
    rows = [(case.get("name"), scenario_status(case), float(case.get("time") or 0))
            for case in root.iter("testcase")]
    if not rows:
        raise ValueError("JUnit не содержит результатов сценариев")
    rows.sort(key=lambda row: -row[2])
    panels = []
    metrics = [
        ("Пройдено", sum(s == "Пройден" for _, s, _ in rows), "short"),
        ("Ошибки", sum(s == "Ошибка" for _, s, _ in rows), "short"),
        ("Пропущено", sum(s == "Пропущен" for _, s, _ in rows), "short"),
        ("Время тестов", float(root.attrib["time"]), "s"),
    ]
    for i, (title, value, unit) in enumerate(metrics):
        panels.append({
            "id": i + 1, "type": "stat", "title": title,
            "gridPos": {"x": i * 6, "y": 0, "w": 6, "h": 4},
            "targets": [query([title], [[value]])],
            "fieldConfig": {"defaults": {"unit": unit}, "overrides": []},
            "options": {"colorMode": "none", "graphMode": "none", "textMode": "value"},
        })
    panels.append({
        "id": 5, "type": "barchart", "title": "Длительность сценариев",
        "gridPos": {"x": 0, "y": 4, "w": 24, "h": 16},
        "targets": [query(["Сценарий", "Время"], [(name, seconds) for name, _, seconds in rows])],
        "fieldConfig": {"defaults": {"unit": "s", "min": 0, "color": {"mode": "fixed", "fixedColor": "blue"}}, "overrides": []},
        "options": {"orientation": "horizontal", "showValue": "always", "legend": {"showLegend": False},
                    "xField": "Сценарий", "xTickLabelMaxLength": 90},
    })
    panels.append({
        "id": 6, "type": "table", "title": "Результаты сценариев",
        "gridPos": {"x": 0, "y": 20, "w": 24, "h": max(13, len(rows) + 2)},
        "targets": [query(["Сценарий", "Результат", "Время, с"], rows)],
        "fieldConfig": {"defaults": {}, "overrides": [
            {"matcher": {"id": "byName", "options": col}, "properties": [{"id": "custom.width", "value": 145}]}
            for col in ("Результат", "Время, с")
        ]},
    })
    for panel in panels:
        panel["datasource"] = SOURCE
    dashboard = {"uid": "lab5-auth", "title": "ЛР 5 · Аутентификация", "schemaVersion": 39,
                 "description": f"Результаты Godog, прогон {report.parent.name}.",
                 "editable": False, "panels": panels, "time": {"from": "now-1h", "to": "now"}}
    shutil.copytree(Path(__file__).resolve().parents[1] / "grafana", output, dirs_exist_ok=True)
    target = output / "dashboards" / "auth.json"
    target.parent.mkdir(exist_ok=True)
    target.write_text(json.dumps(dashboard, ensure_ascii=False, indent=2), encoding="utf-8")


if __name__ == "__main__":
    build(Path(sys.argv[1]), Path(sys.argv[2]))
