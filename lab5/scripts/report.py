import json
import uuid
from pathlib import Path
import xml.etree.ElementTree as ET


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
