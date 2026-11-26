#!/usr/bin/env bash
# Тело git-хука pre-commit. Проверяет снимок индекса (то, что реально уйдёт
# в коммит), а не рабочую копию: неиндексированные правки не влияют на итог.
# Пропустить проверку: git commit --no-verify
set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
MODULES="${LAB7_MODULES:-project/VitaLineClinic lab7}"
SNAPSHOT="$(mktemp -d "${TMPDIR:-/tmp}/lab7-precommit.XXXXXX")"
trap 'rm -rf "$SNAPSHOT"' EXIT

# Выгружаем из индекса только проверяемые модули.
# shellcheck disable=SC2086
git -C "$ROOT" ls-files -z -- $MODULES | git -C "$ROOT" checkout-index -z --stdin --prefix="$SNAPSHOT/"

echo "pre-commit: статический анализ ($MODULES)"
if ! LAB7_REPORTS="$SNAPSHOT/.reports" "$ROOT/lab7/scripts/check.sh" "$SNAPSHOT"; then
	echo
	echo "Коммит отклонён: исправьте замечания или используйте git commit --no-verify" >&2
	exit 1
fi
