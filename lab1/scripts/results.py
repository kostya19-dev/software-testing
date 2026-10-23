import argparse
import json
import xml.etree.ElementTree as ET
from pathlib import Path

from cases import matching_cases, read_cases


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("directory", type=Path)
    parser.add_argument("status", type=int)
    args = parser.parse_args()
    directory = args.directory
    junit = directory / "junit.xml"
    if not junit.exists():
        raise SystemExit(f"JUnit не создан; код запуска: {args.status}")

    root = ET.parse(junit).getroot()
    cases = list(root.iter("testcase"))
    documented = read_cases()
    for case in cases:
        matches = matching_cases(documented, case.get("classname", ""), case.get("name", ""))
        if not matches:
            continue
        properties = case.find("properties")
        if properties is None:
            properties = ET.SubElement(case, "properties")
        for name, field in (("Use case", "usecase"), ("Test case", "id")):
            values = ", ".join(dict.fromkeys(item[field] for item in matches))
            ET.SubElement(properties, "property", name=name, value=values)
    ET.indent(root)
    ET.ElementTree(root).write(junit, encoding="utf-8", xml_declaration=True)
    failed = sum(
        case.find("failure") is not None or case.find("error") is not None
        for case in cases
    )
    skipped = sum(case.find("skipped") is not None for case in cases)
    order = []
    with (directory / "events.jsonl").open() as events:
        for line in events:
            event = json.loads(line)
            if event.get("Action") == "run" and event.get("Test"):
                order.append(f"{event['Package']}\t{event['Test']}")
    (directory / "order.txt").write_text("\n".join(order) + "\n")

    meta = json.loads((directory / "run.json").read_text())
    meta["exit_code"] = args.status
    (directory / "run.json").write_text(
        json.dumps(meta, ensure_ascii=False, indent=2) + "\n"
    )
    properties = {
        "Mode": meta["mode"],
        "Suites": meta["suite"],
        "Seed": meta["seed"],
        "ParallelPackages": meta["parallel_packages"],
        "Go": meta["go_version"],
        "ExitCode": args.status,
    }
    (directory / "environment.properties").write_text(
        "".join(f"{name}={value}\n" for name, value in properties.items())
    )
    print(f"JUnit: {len(cases)} записей; ошибок: {failed}; пропущено: {skipped}")
    print(f"Allure: make show RESULTS={directory}")


if __name__ == "__main__":
    main()
