#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
for tool in gocov gocov-xml gobco; do
  if [[ ! -x ".tools/$tool" ]]; then echo 'Run make setup first' >&2; exit 2; fi
done
mkdir -p reports/coverage
out=reports/coverage
go test -count=1 -covermode=atomic -coverpkg=vitaline-clinic/internal/domain,vitaline-clinic/internal/usecase,vitaline-clinic/internal/storage/memory,vitaline-clinic/internal/storage/mongodb -coverprofile="$out/coverage.out" ./domain ./usecase ./storage/...
go tool cover -func="$out/coverage.out" | tee "$out/statements.txt"
go tool cover -html="$out/coverage.out" -o "$out/source.html"
.tools/gocov convert "$out/coverage.out" | .tools/gocov-xml > "$out/cobertura.xml"
python3 -c 'import xml.etree.ElementTree as E; r=E.parse("reports/coverage/cobertura.xml").getroot(); print("Cobertura entries (not unique source lines): "+r.get("lines-covered")+"/"+r.get("lines-valid")+" ("+str(round(float(r.get("line-rate"))*100,2))+"%)")' | tee "$out/cobertura.txt"
python3 scripts/branches.py
