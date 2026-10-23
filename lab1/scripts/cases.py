import argparse
import os
import re
import subprocess
import xml.etree.ElementTree as ET
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def read_cases():
    cases = []
    usecase = ""
    for line in (ROOT / "docs/spec.md").read_text().splitlines():
        heading = re.fullmatch(r"## (UC-\d+)\. (.+)", line)
        if heading:
            usecase = heading[1] + " " + heading[2]
        match = re.fullmatch(
            r"\| (TC-\d+-\d+) \| (.+?) \| (.+?) \| \[(Test[^]]+)\]\(\.\./([^)]+)\) \|",
            line,
        )
        if not match:
            continue
        case_id, condition, expected, test, filename = match.groups()
        path = (ROOT / filename).resolve()
        if not usecase or not path.is_relative_to(ROOT) or not path.is_file():
            raise ValueError(f"{case_id}: неверная ссылка на тест")
        function = test.split("/")[0]
        if not re.search(r"^func " + re.escape(function) + r"\(", path.read_text(), re.M):
            raise ValueError(f"{case_id}: {function} не найден в {filename}")
        cases.append({
            "id": case_id,
            "usecase": usecase,
            "condition": condition,
            "expected": expected,
            "test": test,
            "file": filename,
            "package": "vitaline-clinic/lab1/" + str(Path(filename).parent),
        })
    if not cases or len({case["id"] for case in cases}) != len(cases):
        raise ValueError("Список тест-кейсов пуст или содержит повторные ID")
    return cases


def matching_cases(cases, package, test):
    return [
        case for case in cases
        if case["package"] == package
        and (test == case["test"] or test.startswith(case["test"] + "/"))
    ]


def check(cases, results):
    tree = ET.parse(results)
    records = {
        (test.get("classname"), test.get("name")): test
        for test in tree.iter("testcase")
    }
    problems = []
    for case in cases:
        record = records.get((case["package"], case["test"]))
        if record is None:
            problems.append(f'{case["id"]}: тест отсутствует в прогоне: {case["test"]}')
        elif any(record.find(tag) is not None for tag in ("failure", "error", "skipped")):
            problems.append(f'{case["id"]}: тест не прошёл: {case["test"]}')
    if problems:
        raise SystemExit("\n".join(problems))
    print(f"Проверено ссылок на пройденные тесты: {len(cases)}. Файлы и имена совпадают.")


def run(cases, case_id):
    case = next((case for case in cases if case["id"] == case_id), None)
    if case is None:
        raise SystemExit(f"Неизвестный тест-кейс: {case_id}. См. docs/spec.md")
    suite = Path(case["file"]).parent.name
    pattern = "/".join("^" + re.escape(part) + "$" for part in case["test"].split("/"))
    print(f'{case["id"]}: {case["usecase"]}', flush=True)
    print(f'Условие: {case["condition"]}\nОжидание: {case["expected"]}', flush=True)
    env = dict(os.environ, CASE=case_id, TEST_PATTERN=pattern)
    result = subprocess.run(
        ["bash", "scripts/run.sh", "case", suite], cwd=ROOT, env=env, check=False
    )
    if result.returncode == 0:
        results = ROOT / "reports/cases" / case_id / "junit.xml"
        check([case], results)
    return result.returncode


def main():
    parser = argparse.ArgumentParser()
    commands = parser.add_subparsers(dest="command", required=True)
    runner = commands.add_parser("run")
    runner.add_argument("case_id")
    checker = commands.add_parser("check")
    checker.add_argument("results", type=Path)
    args = parser.parse_args()
    cases = read_cases()
    if args.command == "check":
        check(cases, args.results)
    else:
        raise SystemExit(run(cases, args.case_id))


if __name__ == "__main__":
    main()
