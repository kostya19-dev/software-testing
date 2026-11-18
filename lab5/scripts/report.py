import html
from pathlib import Path
import xml.etree.ElementTree as ET


def render(directory: Path, exit_code: int, cleanup: int):
    junit = directory / "junit.xml"
    rows = []
    tests = failures = 0
    if junit.exists():
        root = ET.parse(junit).getroot()
        tests = int(root.get("tests", 0))
        failures = int(root.get("failures", 0)) + int(root.get("errors", 0))
        for case in root.findall(".//testcase"):
            failed = case.find("failure")
            message = "" if failed is None else failed.get("message", "")
            status = "PASS" if failed is None and case.get("status") == "passed" else "FAIL"
            rows.append("<tr><td>" + html.escape(case.get("name", "")) + "</td><td class='" + status + "'>"
                        + status + "</td><td>" + html.escape(case.get("time", "")) + "</td><td>"
                        + html.escape(message) + "</td></tr>")
    status = "PASS" if exit_code == 0 and cleanup == 0 and tests > 0 and failures == 0 else "FAIL"
    document = """<!doctype html><html lang="ru"><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>ЛР5 — Аутентификация VitaLine Clinic</title>
<style>body{font:16px/1.55 system-ui,sans-serif;max-width:1150px;margin:40px auto;padding:0 24px;color:#183047;background:#f6f8fb}
h1{font-size:30px}table{width:100%;border-collapse:collapse;background:white}th,td{text-align:left;padding:14px;border-bottom:1px solid #d9e1e9}
th{background:#e8eef5}.PASS{color:#08734d;font-weight:700}.FAIL{color:#b12732;font-weight:700}code{overflow-wrap:anywhere}a{color:#145ead}</style>
<h1>ЛР5 · Аутентификация VitaLine Clinic</h1>"""
    document += f"<p>Прогон <code>{html.escape(directory.name)}</code> · <strong class='{status}'>{status}</strong></p>"
    document += f"<p>Сценариев: {tests}. Ошибок сценариев: {failures}. Очистка стенда: {'успешна' if cleanup == 0 else 'ошибка'}.</p>"
    if not tests:
        document += "<p>BDD не завершён или стенд не подготовлен. Смотрите вывод команды.</p>"
    document += "<table><thead><tr><th>Сценарий</th><th>Результат</th><th>Секунды</th><th>Причина</th></tr></thead><tbody>" + "".join(rows) + "</tbody></table>"
    document += '<p><a href="junit.xml">JUnit</a> · <a href="cucumber.json">Cucumber JSON</a> · <a href="bdd.log">Вывод BDD</a> · <a href="cleanup.txt">Очистка</a></p></html>'
    (directory / "report.html").write_text(document, encoding="utf-8")


if __name__ == "__main__":
    import sys
    render(Path(sys.argv[1]), int(sys.argv[2]), int(sys.argv[3]))
