#!/usr/bin/env bash
set -euo pipefail
printf '%s\t%s\n' "$$" "$1" >> "$LAB_PROCESS_LOG"
exec "$@"
