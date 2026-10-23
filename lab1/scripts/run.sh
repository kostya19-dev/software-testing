#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

mode=${1:-normal}
suite=${2:-${SUITES:-${SUITE:-all}}}
case "$mode" in
  normal|shuffle|offline|errors|case) ;;
  *) echo "Неизвестный режим: $mode" >&2; exit 2 ;;
esac

read -r -a requested <<< "$suite"
if (( ${#requested[@]} == 0 )); then
  echo 'Выберите хотя бы один набор тестов' >&2
  exit 2
fi

packages=()
selected=()
for item in "${requested[@]}"; do
  case "$item" in
    all)
      if (( ${#requested[@]} != 1 )); then
        echo 'all нельзя объединять с отдельными наборами' >&2
        exit 2
      fi
      packages=(./domain ./usecase ./storage/memory ./storage/mongodb)
      selected=(all)
      ;;
    domain|usecase|memory|mongodb)
      if [[ " ${selected[*]} " == *" $item "* ]]; then
        continue
      fi
      selected+=("$item")
      case "$item" in
        domain|usecase) packages+=("./$item") ;;
        memory|mongodb) packages+=("./storage/$item") ;;
      esac
      ;;
    *) echo 'Доступные наборы: domain, usecase, memory, mongodb, all' >&2; exit 2 ;;
  esac
done

suite="${selected[*]}"
slug=${suite// /-}
seed=${SEED:-off}
parallel=${PARALLEL_PACKAGES:-1}
if [[ ! "$parallel" =~ ^[1-9][0-9]*$ ]]; then
  echo 'PARALLEL_PACKAGES должно быть положительным целым числом' >&2
  exit 2
fi
if [[ "$mode" == shuffle && "$seed" == off ]]; then
  seed=on
fi
if [[ "$seed" == on ]]; then
  seed=$(python3 -c 'import secrets; print(secrets.randbits(63))')
fi
if [[ "$seed" != off ]]; then
  if [[ ! "$seed" =~ ^[0-9]+$ ]] || ! python3 -c 'import sys; sys.exit(not 0 <= int(sys.argv[1]) < 2**63)' "$seed"; then
    echo 'SEED: off, on или целое число от 0 до 9223372036854775807' >&2
    exit 2
  fi
  if [[ "$mode" == normal ]]; then
    mode=shuffle
  fi
fi
if [[ ! -x .tools/gotestsum ]]; then
  echo 'Сначала выполните make setup' >&2
  exit 2
fi

out="$PWD/reports/$mode"
if [[ "$mode" == case ]]; then
  [[ "${CASE:-}" =~ ^TC-[0-9]+-[0-9]+$ ]] || { echo "Неверный CASE" >&2; exit 2; }
  out="$PWD/reports/cases/$CASE"
elif [[ "$suite" != all ]]; then
  out="$out/$slug"
fi
mkdir -p "$out"
export LAB_PROCESS_LOG="$out/processes.tsv"
: > "$LAB_PROCESS_LOG"

flags=(-count=1 -timeout=120s -p "$parallel" -exec "$PWD/scripts/test-process.sh")
if [[ "$seed" != off ]]; then
  flags+=(-shuffle="$seed")
fi
if [[ -n "${TEST_PATTERN:-}" ]]; then
  flags+=(-run="$TEST_PATTERN")
fi
version=$(go version)
python3 - "$out" "$mode" "$suite" "$seed" "$parallel" "$version" <<'PY'
import datetime
import os
import json
import sys
from pathlib import Path

directory, mode, suite, seed, parallel, version = sys.argv[1:]
out = Path(directory)
for name in ("junit.xml", "events.jsonl", "order.txt"):
    (out / name).unlink(missing_ok=True)
meta = {
    "mode": mode,
    "case": os.getenv("CASE", ""),
    "test_pattern": os.getenv("TEST_PATTERN", ""),
    "suite": suite,
    "seed": seed,
    "parallel_packages": int(parallel),
    "go_version": version,
    "started_at": datetime.datetime.now().astimezone().isoformat(),
}
(out / "run.json").write_text(json.dumps(meta, ensure_ascii=False, indent=2) + "\n")
PY

target=$mode
if [[ "$mode" == normal ]]; then
  target=test
fi
if [[ "$mode" == case ]]; then
  printf '#!/usr/bin/env bash\nset -euo pipefail\ncd %q\nexec make case CASE=%q SEED=%q PARALLEL_PACKAGES=%q\n' \
    "$PWD" "$CASE" "$seed" "$parallel" > "$out/replay.sh"
else
printf '#!/usr/bin/env bash\nset -euo pipefail\ncd %q\nexec make %q SUITES=%q SEED=%q PARALLEL_PACKAGES=%q\n' \
  "$PWD" "$target" "$suite" "$seed" "$parallel" > "$out/replay.sh"
fi

if [[ "$mode" == offline ]]; then
  python3 - <<'PY' > "$out/network.txt"
import os
import socket

with socket.socket() as connection:
    if connection.connect_ex(("198.51.100.1", 443)) == 0:
        raise SystemExit("Внешняя сеть доступна")
interfaces = socket.if_nameindex()
if [name for _, name in interfaces] != ["lo"]:
    raise SystemExit(f"Лишние сетевые интерфейсы: {interfaces}")
print("Network namespace:", os.readlink("/proc/self/ns/net"))
print("Interfaces:", interfaces)
PY
fi

echo "Suite: $suite | mode: $mode | seed: $seed"
set +e
.tools/gotestsum --format pkgname --junitfile "$out/junit.xml" --jsonfile "$out/events.jsonl" \
  -- "${flags[@]}" "${packages[@]}" 2>&1 | tee "$out/console.txt"
status=${PIPESTATUS[0]}
set -e
python3 scripts/results.py "$out" "$status"
report_status=0
python3 scripts/report.py "$out" || report_status=$?
if (( status != 0 )); then
  exit "$status"
fi
exit "$report_status"
