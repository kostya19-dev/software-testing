#!/usr/bin/env bash
set -euo pipefail

LAB7="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ROOT="$(cd "${1:-$LAB7/..}" && pwd)"
MODULES="${LAB7_MODULES:-project/VitaLineClinic lab7}"
MAX_CYCLOMATIC="${MAX_CYCLOMATIC:-10}"
MAX_VOLUME="${MAX_VOLUME:-3000}"
MAX_DIFFICULTY="${MAX_DIFFICULTY:-30}"
REPORTS="${LAB7_REPORTS:-$LAB7/reports}"

source "$LAB7/scripts/tools.sh"

failed=()
step() {
	local name="$1"
	shift
	printf '\n==> %s\n' "$name"
	if "$@"; then
		printf '    ok\n'
	else
		printf '    FAILED: %s\n' "$name"
		failed+=("$name")
	fi
}

mkdir -p "$REPORTS"
for module in $MODULES; do
	dir="$ROOT/$module"
	[ -f "$dir/go.mod" ] || { echo "skip $module: нет go.mod"; continue; }
	slug="${module//\//-}"
	step "$module: golangci-lint (gofmt, goimports, govet, staticcheck, revive, gocyclo<=$MAX_CYCLOMATIC, ...)" \
		bash -c 'cd "$1" && "$2" run ./...' _ "$dir" "$GOLANGCI_LINT"
	step "$module: цикломатическая сложность и метрики Холстеда" \
		"$COMPLEXITY" -dir "$dir" -top 10 \
		-max-cyclomatic "$MAX_CYCLOMATIC" -max-volume "$MAX_VOLUME" -max-difficulty "$MAX_DIFFICULTY" \
		-json "$REPORTS/complexity-$slug.json" ${GITHUB_STEP_SUMMARY:+-markdown "$GITHUB_STEP_SUMMARY"}
done

if [ ${#failed[@]} -gt 0 ]; then
	printf '\nСтатический анализ не пройден:\n'
	printf '  - %s\n' "${failed[@]}"
	exit 1
fi
printf '\nСтатический анализ пройден\n'
