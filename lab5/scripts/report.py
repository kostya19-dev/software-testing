import json
import os
import uuid
from pathlib import Path
import xml.etree.ElementTree as ET


def write_trend(directory: Path):
    target = directory / "allure-results"
    history = target / "history"
    history.mkdir(parents=True, exist_ok=True)
    points = []
    if os.environ.get("GITHUB_RUN_NUMBER"):
        build = int(os.environ["GITHUB_RUN_NUMBER"])
        archive = Path(os.environ["RUNNER_TEMP"], "lab5-history")
        records = []
        for run in archive.glob("*"):
            reports = sorted(run.rglob("junit.xml"))
            if reports:
                records.append((int(run.name), reports[-1]))
        title = f"GitHub Actions #{build}"
    else:
        previous = sorted(path for path in directory.parent.glob("*/junit.xml") if path.parent.name < directory.name)
        records = list(enumerate(previous, 1))
        build = len(records) + 1
        title = directory.name
    for order, path in sorted(records, reverse=True)[:19]:
        suite = ET.parse(path).getroot()
        total = int(suite.attrib["tests"])
        if total == 0:
            continue
        failed = int(suite.get("failures", 0))
        broken = int(suite.get("errors", 0))
        skipped = int(suite.get("skipped", 0))
        points.append({"buildOrder": order, "reportName": path.parent.name,
                       "data": {"passed": total - failed - broken - skipped, "failed": failed,
                                "broken": broken, "skipped": skipped, "unknown": 0, "total": total}})
    (history / "history-trend.json").write_text(json.dumps(points), encoding="utf-8")
    print(f"Allure trend: {len(points)} previous runs and the current run")
    (target / "executor.json").write_text(json.dumps({"name": "GitHub Actions" if os.getenv("GITHUB_ACTIONS") else "Local",
                                                     "reportName": title, "buildOrder": build}), encoding="utf-8")


def render(directory: Path, started_ns: int):
    cases = {case.get("name"): case for case in ET.parse(directory / "junit.xml").iter("testcase")}
    target = directory / "allure-results"
    target.mkdir(exist_ok=True)
    features = json.loads((directory / "cucumber.json").read_text(encoding="utf-8"))
    for feature in features:
        for scenario in feature["elements"]:
            case = cases.pop(scenario["name"])
            start = started_ns
            steps = []
            for step in scenario["steps"]:
                result = step["result"]
                end = started_ns + int(result.get("duration") or 0)
                status = result["status"]
                if status not in ("passed", "failed", "skipped"):
                    status = "broken"
                steps.append({"name": (step["keyword"] + step["name"]).strip(), "status": status,
                              "statusDetails": {"message": result.get("error_message", "")},
                              "start": started_ns // 1_000_000, "stop": end // 1_000_000})
                started_ns = end
            failure = case.find("failure")
            error = case.find("error")
            status = case.get("status", "passed")
            if failure is not None:
                status = "failed"
            elif error is not None:
                status = "broken"
            elif case.find("skipped") is not None:
                status = "skipped"
            problem = failure if failure is not None else error
            uid = str(uuid.uuid4())
            started_ns = max(started_ns, start + round(float(case.get("time", 0)) * 1_000_000_000))
            data = {"uuid": uid, "historyId": scenario["id"], "fullName": scenario["id"],
                    "name": scenario["name"], "status": status, "steps": steps,
                    "statusDetails": {"message": problem.get("message", "") if problem is not None else ""},
                    "labels": [{"name": "suite", "value": feature["name"]}],
                    "start": start // 1_000_000, "stop": started_ns // 1_000_000}
            (target / f"{uid}-result.json").write_text(json.dumps(data, ensure_ascii=False), encoding="utf-8")
    if cases or not any(target.glob("*-result.json")):
        raise ValueError("Incomplete Godog scenario results")
    write_trend(directory)
