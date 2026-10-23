import argparse
import hashlib
import json
import subprocess
import tempfile
import uuid
import xml.etree.ElementTree as ET
from datetime import datetime
from pathlib import Path

from cases import ROOT, matching_cases, read_cases


def read_times(directory):
    times = {}
    for line in (directory / "events.jsonl").read_text().splitlines():
        event = json.loads(line)
        if not event.get("Test") or event["Action"] not in ("run", "pass", "fail", "skip"):
            continue
        key = (event["Package"], event["Test"])
        field = "start" if event["Action"] == "run" else "stop"
        times.setdefault(key, {})[field] = int(
            datetime.fromisoformat(event["Time"].replace("Z", "+00:00")).timestamp() * 1000
        )
    return times


def export_results(directory, target):
    documented = read_cases()
    records = list(ET.parse(directory / "junit.xml").iter("testcase"))
    times = read_times(directory)
    keys = {(record.get("classname"), record.get("name")) for record in records}
    sources = {}
    for record in records:
        package, test = record.get("classname"), record.get("name")
        group = any(p == package and name.startswith(test + "/") for p, name in keys)
        matches = matching_cases(documented, package, test)
        if not matches and not group:
            raise ValueError(f"Нет описания в docs/spec.md: {package}/{test}")

        primary = max(matches, key=lambda case: len(case["test"]), default=None)
        identity = package + "/" + test
        result = {
            "uuid": str(uuid.uuid4()),
            "historyId": hashlib.sha256(identity.encode()).hexdigest(),
            "fullName": identity,
            "name": "Группа тестов: " + test if group else (
                f'{primary["id"]}: {primary["condition"]}'
            ),
            "status": "passed",
            "stage": "finished",
            "labels": [
                {"name": "suite", "value": package.removeprefix("vitaline-clinic/lab1/")},
                {"name": "package", "value": package},
                {"name": "language", "value": "go"},
                {"name": "framework", "value": "testing"},
            ],
            "parameters": [{"name": "Go-тест", "value": test}],
            "attachments": [],
            **times.get((package, test), {}),
        }
        for tag, status in (("skipped", "skipped"), ("failure", "failed"), ("error", "broken")):
            failure = record.find(tag)
            if failure is not None:
                result["status"] = status
                result["statusDetails"] = {
                    "message": failure.get("message", ""),
                    "trace": failure.text or "",
                }
        description = []
        if group:
            result["labels"].append({"name": "feature", "value": "Группы тестов"})
            description.append(
                "Результат родительской Go-функции. Отдельные сценарии находятся "
                "в разделах UC. Эта запись не считается самостоятельным сценарием."
            )
        for case in matches:
            if not group and case is primary:
                for name, value in (
                    ("feature", case["usecase"]),
                    ("story", case["id"] + ": " + case["condition"]),
                ):
                    label = {"name": name, "value": value}
                    if label not in result["labels"]:
                        result["labels"].append(label)
            description.append(
                f'### {case["usecase"]}\n\n'
                f'**Тест-кейс:** {case["id"]}\n\n'
                f'**Условие:** {case["condition"]}\n\n'
                f'**Ожидаемый результат:** {case["expected"]}\n\n'
                f'**Файл:** `{case["file"]}`\n\n'
                f'**Тест из документации:** `{case["test"]}`\n\n'
                f'**Запустить из lab1:** `make case CASE={case["id"]}`'
            )
            filename = case["file"]
            if filename not in sources:
                source = str(uuid.uuid4()) + "-attachment.txt"
                (target / source).write_text((ROOT / filename).read_text())
                sources[filename] = source
            attachment = {
                "name": "Исходный код: " + filename,
                "source": sources[filename],
                "type": "text/plain",
            }
            if attachment not in result["attachments"]:
                result["attachments"].append(attachment)
        result["description"] = "\n\n".join(description)
        for tag in ("system-out", "system-err"):
            output = record.findtext(tag)
            if output:
                source = str(uuid.uuid4()) + "-attachment.txt"
                (target / source).write_text(output)
                result["attachments"].append({
                    "name": "Вывод теста: " + tag, "source": source, "type": "text/plain",
                })
        (target / (result["uuid"] + "-result.json")).write_text(
            json.dumps(result, ensure_ascii=False)
        )
    environment = directory / "environment.properties"
    if environment.exists():
        (target / environment.name).write_bytes(environment.read_bytes())


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("directory", type=Path)
    args = parser.parse_args()
    directory = args.directory.resolve()
    allure = ROOT / ".tools/allure-cli/node_modules/.bin/allure"
    if not allure.is_file():
        raise SystemExit("Allure не установлен: выполните make setup")
    output = directory / "allure"
    with tempfile.TemporaryDirectory(prefix="allure-") as temporary:
        target = Path(temporary)
        export_results(directory, target)
        subprocess.run([
            str(allure), "generate", str(target), "--clean", "--single-file",
            "-o", str(output),
        ], check=True)
    page = output / "index.html"
    html = page.read_text()
    html = html.replace('lang="en"', 'lang="ru"', 1)
    html = html.replace("<title>Allure Report</title>", "<title>ЛР1 — отчёт Allure</title>", 1)
    html = html.replace("<head>", """<head>
<script>
try {
    const key = "ALLURE_REPORT_SETTINGS";
    const settings = JSON.parse(localStorage.getItem(key) || "{}");
    settings.language = "ru";
    localStorage.setItem(key, JSON.stringify(settings));
} catch (_) {}
</script>""", 1)
    page.write_text(html)
    print(f"Готовый HTML: {page}", flush=True)


if __name__ == "__main__":
    main()
