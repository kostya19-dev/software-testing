import datetime
import json
import signal
import subprocess
import time
from pathlib import Path

lab = Path(__file__).resolve().parents[1]
run_id = "interrupt-" + datetime.datetime.now().strftime("%Y%m%d-%H%M%S")
(lab / "reports").mkdir(exist_ok=True)
log = (lab / "reports" / f"{run_id}.log").open("w")
process = subprocess.Popen(
    ["python3", str(lab / "scripts/pipeline.py"), "all", "--id", run_id],
    stdout=log,
    stderr=subprocess.STDOUT,
)
statefile = lab / ".runs" / run_id / "state.json"
try:
    for _ in range(600):
        if process.poll() is not None:
            raise RuntimeError("Run ended before interruption")
        if statefile.exists():
            try:
                state = json.loads(statefile.read_text())
            except json.JSONDecodeError:
                time.sleep(0.5)
                continue
            if state.get("stages", {}).get("e2e") == "running":
                subprocess.run(
                    [
                        "docker",
                        "exec",
                        "tiopo-lab2-mongo-1",
                        "mongosh",
                        "--quiet",
                        "--eval",
                        'db.getSiblingDB(' + json.dumps(state["database"]) + ').interruption_probe.insertOne({_id:"must-disappear"})',
                    ],
                    check=True,
                    stdout=log,
                    stderr=log,
                )
                process.send_signal(signal.SIGTERM)
                break
        time.sleep(0.5)
    else:
        raise RuntimeError("Timed out waiting for E2E")
    code = process.wait(timeout=120)
    state = json.loads(statefile.read_text())
    assert code != 0 and state["cleaned"] and state["stages"]["e2e"] == "failed", state
    rollback = (lab / "reports" / run_id / "rollback.txt").read_text()
    assert "FAIL" not in rollback and "finalize: PASS" in rollback, rollback
    (lab / "reports" / run_id / "interruption-proof.txt").write_text(
        "SIGTERM sent during E2E preparation after inserting a probe document.\nNonzero exit observed; rollback matched baseline; session removed; persistent stand preserved.\n"
    )
    print("Verified interruption:", run_id)
finally:
    if process.poll() is None:
        process.terminate()
        process.wait(timeout=120)
    log.close()
