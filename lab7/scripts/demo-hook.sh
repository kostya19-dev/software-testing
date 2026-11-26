#!/usr/bin/env bash
set -euo pipefail

LAB7="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SRC="$(cd "$LAB7/.." && pwd)"
DEMO="$(mktemp -d "${TMPDIR:-/tmp}/lab7-demo.XXXXXX")"
trap 'rm -rf "$DEMO"' EXIT

copy() { mkdir -p "$DEMO/$(dirname "$1")" && cp -r "$SRC/$1" "$DEMO/$1"; }
for path in .githooks lab7/go.mod lab7/.golangci.yml lab7/cmd lab7/internal lab7/scripts \
	project/VitaLineClinic/go.mod project/VitaLineClinic/go.sum project/VitaLineClinic/.golangci.yml \
	project/VitaLineClinic/cmd project/VitaLineClinic/internal project/VitaLineClinic/migrations; do
	copy "$path"
done
[ -d "$LAB7/.tools" ] && ln -s "$LAB7/.tools" "$DEMO/lab7/.tools"

cd "$DEMO"
git init --quiet
git config user.name "lab7 demo"
git config user.email "lab7@example.invalid"
git config core.hooksPath .githooks
printf '.tools\nreports/\n' > lab7/.gitignore

banner() { printf '\n\033[1m######## %s\033[0m\n' "$1"; }

banner "1. Исходный код проходит проверки — коммит создаётся"
git add -A
git commit --quiet -m "Исходный код"
git log --oneline

banner "2. Добавлена функция со сложностью 11 — хук отклоняет коммит"
cat > project/VitaLineClinic/internal/domain/demo_complexity.go <<'EOF'
package domain

func Grade(score int) string {
	switch {
	case score > 90:
		return "A"
	case score > 80:
		return "B"
	case score > 70:
		return "C"
	case score > 60:
		return "D"
	case score > 50:
		return "E"
	case score > 40:
		return "F"
	case score > 30:
		return "G"
	case score > 20:
		return "H"
	case score > 10:
		return "I"
	case score > 0:
		return "J"
	}
	return "-"
}
EOF
git add -A
if git commit --quiet -m "Сложная функция"; then
	echo "ОШИБКА: хук пропустил коммит" >&2
	exit 1
fi
echo "Коммит не создан, последний коммит:"
git log --oneline -1

banner "3. git commit --no-verify пропускает хук"
git commit --quiet --no-verify -m "Сложная функция (--no-verify)"
git log --oneline
