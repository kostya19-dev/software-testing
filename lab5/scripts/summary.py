import html
from pathlib import Path
import re
import sys
import xml.etree.ElementTree as ET


def summary(directory):
    lines = ["## Аутентификация VitaLine Clinic", ""]
    junit = directory / "junit.xml"
    try:
        cases = ET.parse(junit).getroot().findall(".//testcase")
    except (FileNotFoundError, ET.ParseError):
        cases = []
    if not cases:
        lines.append("Нет результатов сценариев. Подробности в шаге **Run isolated authentication scenarios**.")
        return "\n".join(lines)

    rows = []
    for case in cases:
        status = case.get("status")
        if case.find("failure") is not None or case.find("error") is not None and status != "skipped":
            status = "failed"
        elif case.find("skipped") is not None:
            status = "skipped"
        name = html.escape(" ".join(case.get("name", "").split())).replace("|", r"\|")
        rows.append((status, name, float(case.get("time") or 0)))

    passed = sum(status == "passed" for status, _, _ in rows)
    skipped = sum(status == "skipped" for status, _, _ in rows)
    lines.append(f"**Сценарии:** ✅ {passed} прошли · ❌ {len(rows) - passed - skipped} с ошибкой · ⏭ {skipped} пропущены")
    log = directory / "bdd.log"
    if log.is_file():
        text = re.sub(r"\x1b\[[0-9;]*m", "", log.read_text(encoding="utf-8", errors="replace"))
        steps = re.search(r"(?m)^(\d+) steps? \(([^)\n]+)\)", text)
        if steps:
            ok = re.search(r"(\d+) passed", steps[2])
            lines.extend(["", f"**Успешные шаги:** {ok[1] if ok else 0} из {steps[1]}"])
    cleanup = directory / "cleanup.txt"
    if cleanup.is_file():
        removed = cleanup.read_text().strip() == "isolated stand removed"
        lines.extend(["", "**Стенд:** " + ("удалён" if removed else "ошибка очистки")])

    lines.extend(["", "| Сценарий | Результат | Время, с | Длительность |", "|---|---|---:|---|"])
    longest = max(seconds for _, _, seconds in rows)
    for status, name, seconds in sorted(rows, key=lambda row: -row[2]):
        blocks = max(1, round(10 * seconds / longest)) if longest and seconds else 0
        label = {"passed": "✅ Прошёл", "skipped": "⏭ Пропущен"}.get(status, "❌ Ошибка")
        lines.append(f"| {name} | {label} | {seconds:.3f} | {'▰' * blocks} |")
    return "\n".join(lines)


if __name__ == "__main__":
    print(summary(Path(sys.argv[1])))
