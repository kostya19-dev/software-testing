#!/usr/bin/env bash
set -euo pipefail
python3 scripts/stand.py up
prefix="parallel-$(date +%Y%m%d-%H%M%S)-$RANDOM"
stand=$(docker inspect --format '{{.Id}}' tiopo-lab2-mongo-1)
mkdir -p reports
python3 scripts/pipeline.py all --id "$prefix-a" > "reports/$prefix-a.log" 2>&1 &
a=$!
python3 scripts/pipeline.py all --id "$prefix-b" > "reports/$prefix-b.log" 2>&1 &
b=$!
trap 'kill -TERM "$a" "$b" 2>/dev/null || true; wait || true' INT TERM
status=0
wait "$a" || status=1
wait "$b" || status=1
if (( status == 0 )); then
  python3 - "$prefix" "$stand" <<'PY'
import json
import subprocess
import sys
from pathlib import Path

prefix, original = sys.argv[1:]
states = [json.loads((Path("reports") / (prefix + suffix) / "summary.json").read_text()) for suffix in ("-a", "-b")]
assert states[0]["database"] != states[1]["database"], "Базы совпали"
assert all(s["cleaned"] and all(v == "passed" for v in s["stages"].values()) for s in states)
current = subprocess.check_output(["docker", "inspect", "--format", "{{.Id}}", "tiopo-lab2-mongo-1"], text=True).strip()
assert current == original, "Общая MongoDB была пересоздана"
print("Оба прогона прошли. Базы разные; постоянная MongoDB сохранилась.")
PY
fi
echo "Запуски: $prefix-a и $prefix-b. Отчёты: http://localhost:8093"
exit "$status"
