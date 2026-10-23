#!/usr/bin/env bash
set -euo pipefail
count=${COUNT:-5}
[[ "$count" =~ ^[1-9][0-9]*$ ]] || { echo 'COUNT должен быть положительным целым числом'; exit 2; }
prefix="parallel-$(date +%Y%m%d-%H%M%S)-$RANDOM"
stand=$(python3 scripts/stand.py snapshot)
mkdir -p reports
pids=()
for ((i=1; i<=count; i++)); do
  DEVELOPER_ID="developer-$i" python3 scripts/pipeline.py all --id "$prefix-$i" </dev/null > "reports/$prefix-$i.log" 2>&1 &
  pids+=("$!")
done
trap 'kill -TERM "${pids[@]}" 2>/dev/null || true; wait || true; exit 130' INT TERM
echo "Запущено прогонов: $count. Логи: make parallel-logs"
status=0
for i in "${!pids[@]}"; do
  if wait "${pids[$i]}"; then
    echo "Прогон $((i+1)): PASS"
  else
    echo "Прогон $((i+1)): FAIL — см. reports/$prefix-$((i+1)).log"
    status=1
  fi
done
if (( status == 0 )); then
  python3 - "$prefix" "$stand" "$count" <<'PY'
import json
import subprocess
import sys
from pathlib import Path

prefix, original, count = sys.argv[1:]
states = [json.loads((Path("reports") / f"{prefix}-{i}" / "summary.json").read_text()) for i in range(1, int(count) + 1)]
assert len({s["database"] for s in states}) == len(states), "Базы совпали"
assert all(s["cleaned"] and all(v == "passed" for v in s["stages"].values()) for s in states)
assert max(s["started"] for s in states) < min(s["finished"] for s in states), "Прогоны не пересеклись во времени"
current = subprocess.check_output(["python3", "scripts/stand.py", "snapshot"], text=True).strip()
assert json.loads(current) == json.loads(original), "Сервисы стенда были пересозданы или перезапущены"
print(f"Все {count} прогонов прошли и пересеклись во времени. Базы разные; сервисы стенда не запускались и не перезапускались.")
PY
fi
echo "Запуски: $prefix-1 ... $prefix-$count. Отчёты: http://localhost:8093"
exit "$status"
