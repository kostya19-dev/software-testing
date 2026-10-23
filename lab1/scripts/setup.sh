#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

for tool in go java npm; do
  command -v "$tool" >/dev/null || { echo "Не найден $tool" >&2; exit 2; }
done

mkdir -p .tools
go mod download
(
  cd tools/coverage
  go mod download
  go build -o ../../.tools/gobco github.com/rillig/gobco
  go build -o ../../.tools/gocov github.com/axw/gocov/gocov
  go build -o ../../.tools/gocov-xml github.com/AlekSi/gocov-xml
  go build -o ../../.tools/gotestsum gotest.tools/gotestsum
)
npm install --prefix .tools/allure-cli --no-audit --no-fund --no-save allure-commandline@2.34.1
