"""Run the lab checks and build a standalone Allure report."""
import argparse
import base64
import datetime as dt
import json
import os
import re
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import time
import unittest
import uuid
from xml.etree import ElementTree as ET

from report import read_summary, chart_source

ROOT = Path(__file__).resolve().parents[1]
TOOLS = ROOT.parent / "lab1/.tools"


class PythonResults(unittest.TextTestResult):
    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self.xml = ET.Element("testsuite", name="Python")
        self.started = {}

    def startTest(self, test):
        self.started[test] = time.monotonic()
        super().startTest(test)

    def stopTest(self, test):
        super().stopTest(test)
        classname, _, name = test.id().rpartition(".")
        case = ET.SubElement(self.xml, "testcase", classname=classname, name=name,
                             time=f"{time.monotonic() - self.started.pop(test):.6f}")
        for tag, entries in (("failure", self.failures), ("error", self.errors),
                             ("skipped", self.skipped), ("skipped", self.expectedFailures)):
            for failed, detail in entries:
                if failed is test or getattr(failed, "test_case", None) is test:
                    ET.SubElement(case, tag, message=str(detail).splitlines()[-1]).text = str(detail)
        if test in self.unexpectedSuccesses:
            ET.SubElement(case, "failure", message="Unexpected success")


def record(raw, name, command, attachments=()):
    start = time.time_ns() // 1_000_000
    proc = subprocess.run(command, cwd=ROOT, capture_output=True, text=True)
    log = proc.stdout + proc.stderr
    print(log, end="")
    uid = str(uuid.uuid4())
    case = {"uuid":uid, "name":name, "fullName":name, "historyId":name,
            "status":"passed" if proc.returncode == 0 else "failed",
            "start":start, "stop":time.time_ns() // 1_000_000,
            "labels":[{"name":"suite","value":"Lab checks"}], "attachments":[]}
    if proc.returncode:
        case["statusDetails"] = {"message":f"Exit code {proc.returncode}", "trace":log}
    (raw / (uid+".txt")).write_text(log)
    case["attachments"].append({"name":"Check output","source":uid+".txt","type":"text/plain"})
    for path, mime in attachments:
        if path.is_file():
            filename = uid+"-"+path.name
            shutil.copyfile(path, raw/filename)
            case["attachments"].append({"name":path.name,"source":filename,"type":mime})
    (raw / (uid+"-result.json")).write_text(json.dumps(case, ensure_ascii=False))
    return proc.returncode == 0


def export_benchmarks(raw, result):
    summary = read_summary(result)
    mode = "Smoke run" if summary["mode"] == "smoke" else "Full experiment"
    manifest = json.loads((result/"manifest.json").read_text())
    inputs = {info.get("format",info["visits"]):info for info in manifest.get("inputs",[])}
    for row in summary["rows"]:
        codec, scenario, size = row["codec"], row["scenario"], row["size"]
        depth = inputs[size]["depth"] if size in inputs else None
        display = f"{size} JSON · depth 100" if isinstance(size,str) else f"{size} records"
        case_id = f"go/{codec}/{scenario}/{size}" + (f"/depth-{depth}" if depth else "")
        uid = str(uuid.uuid4())
        chart = f"{scenario}-{size}.png"
        shutil.copyfile(chart_source(result, scenario, size), raw / chart)
        metrics = uid + "-metrics.json"
        (raw / metrics).write_text(json.dumps(row, ensure_ascii=False, indent=2))
        quantiles = row["percentiles_ns"]
        counts = row["phases"]
        case = {
            "uuid": uid, "name": f"{codec.upper()} · {scenario} · {display}",
            "fullName": case_id, "historyId": case_id, "status": "passed",
            "description": (
                f"{mode}, {summary['run_id']}. {row['trials']} Docker trials, "
                f"{row['requests']} individual latency samples. "
                "Status reflects execution and data validation. "
                "Latency is reported without a pass/fail threshold.\n\n"
                + ", ".join(f"{key} = {value} ns" for key, value in quantiles.items())
                + "\n\n" + ", ".join(f"{phase}: {v['operations']} operations" for phase, v in counts.items())
            ),
            "labels": [{"name": "parentSuite", "value": "Lab 3"},
                       {"name": "suite", "value": "Go microbenchmarks in Docker"},
                       {"name": "subSuite", "value": mode}],
            "parameters": [{"name": "Codec", "value": codec}, {"name": "Input", "value": str(size)}],
            "attachments": [{"name": "JSON and gob charts", "source": chart, "type": "image/png"},
                            {"name": "Scenario metrics", "source": metrics, "type": "application/json"}],
        }
        if size in inputs:
            input_file = result / Path(inputs[size]["file"]).name
            shutil.copyfile(input_file, raw / input_file.name)
            case["attachments"].append({"name":f"Input JSON (depth {depth})", "source":input_file.name,
                                        "type":"application/json"})
            case["description"] += f"\n\nInput: {input_file.name}, JSON depth {depth}."
        (raw / (uid + "-result.json")).write_text(json.dumps(case, ensure_ascii=False))


def trend_history(run):
    previous = []
    for page in sorted(run.parent.glob("*/report/index.html")):
        folder = page.parent.parent
        if folder.name >= run.name:
            continue
        report = page.read_text(encoding="utf-8")
        match = re.search(r"d\('widgets/summary\.json','([^']+)'\)", report)
        if match is None:
            raise ValueError(f"Missing summary widget in {page}")
        widget = json.loads(base64.b64decode(match.group(1)))
        previous.append({"buildOrder": len(previous) + 1, "reportName": f"Run {len(previous) + 1}",
                         "data": widget["statistic"]})
    return len(previous) + 1, previous[-19:][::-1]


def generate_report(allure, run):
    raw = run/"allure-results"
    order, previous = trend_history(run)
    history = raw/"history"
    history.mkdir(exist_ok=True)
    (history/"history-trend.json").write_text(json.dumps(previous))
    started = dt.datetime.strptime(run.name[:15], "%Y%m%d-%H%M%S")
    title = f"Lab 3 - {run.name[:15]}"
    executor = {"name": "Docker on WSL", "type": "local", "buildOrder": order,
                "buildName": f"Run {order} · {started:%H:%M}", "reportName": title}
    (raw/"executor.json").write_text(json.dumps(executor))
    with tempfile.TemporaryDirectory(dir=run) as staging:
        subprocess.run([allure, "generate", str(raw), "--single-file", "-o", staging], check=True)
        page = Path(staging)/"index.html"
        html = page.read_text(encoding="utf-8")
        language = """<script>
try {
  const key = "ALLURE_REPORT_SETTINGS";
  const settings = JSON.parse(localStorage.getItem(key) || "{}") || {};
  settings.language = "en";
  localStorage.setItem(key, JSON.stringify(settings));
} catch (_) {}
</script>"""
        overview = "<style>.executor-icon { display: none !important; }</style>"
        html = html.replace("<head>", "<head>" + language, 1)
        page.write_text(html.replace("</head>", overview + "</head>", 1), encoding="utf-8")
        output = run/"report"
        output.mkdir(exist_ok=True)
        page.replace(output/"index.html")
    print(f"ALLURE_REPORT={output/'index.html'}")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--result", type=Path, default=ROOT/"evidence")
    result = parser.parse_args().result.resolve()
    allure = shutil.which("allure") or str(TOOLS/"allure-cli/node_modules/.bin/allure")
    gotestsum = shutil.which("gotestsum") or str(TOOLS/"gotestsum")
    if not all(os.access(tool, os.X_OK) for tool in (allure,gotestsum)):
        sys.exit("Установите инструменты: make -C ../lab1 setup")
    parent = ROOT/"results/allure"
    parent.mkdir(parents=True, exist_ok=True)
    run = Path(tempfile.mkdtemp(prefix=dt.datetime.now().strftime("%Y%m%d-%H%M%S-"),dir=parent))
    raw = run/"allure-results"
    raw.mkdir()
    go = subprocess.run([gotestsum,"--junitfile",str(raw/"go.xml"),"--","-count=1","./..."],cwd=ROOT)
    if go.returncode and not (raw/"go.xml").is_file():
        print("Go-тесты не сформировали JUnit", file=sys.stderr)
    suite = unittest.defaultTestLoader.discover(str(ROOT/"scripts"),pattern="test_*.py")
    tests = unittest.TextTestRunner(verbosity=2, resultclass=PythonResults).run(suite)
    ET.ElementTree(tests.xml).write(raw/"python.xml", encoding="utf-8",xml_declaration=True)
    vet = record(raw,"Go vet",["go","vet","./..."])
    benchmark = record(raw,"Validate experiment "+result.name,
                       [sys.executable,"scripts/verify.py",str(result)],
                       [(result/"report.html","text/html"),(result/"statistics.csv","text/csv"),
                        (result/"summary.json","application/json")])
    if benchmark:
        export_benchmarks(raw, result)
    generate_report(allure, run)
    return 0 if go.returncode==0 and tests.wasSuccessful() and vet and benchmark else 1


if __name__ == "__main__":
    sys.exit(main())
