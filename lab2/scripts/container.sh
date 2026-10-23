#!/usr/bin/env bash
set -euo pipefail

stage=${1:?stage required}
if [[ "$stage" == api-stand ]]; then
  mkdir -p /src
  cp -a /workspace/. /src/
  cd /src
else
  git clone --quiet /input/source.bundle /src
  cd /src
  git rev-parse HEAD
fi

case "$stage" in
  api|api-stand)
    [[ "$stage" == api-stand ]] || cd project/VitaLineClinic
    go mod download
    go build -o /tmp/clinic ./cmd/clinic
    export APP_CONFIG_PATH=/tmp/config.json VITALINE_LOG_FILE=/tmp/clinic.log
    exec sh scripts/docker-entrypoint.sh /tmp/clinic api
    ;;
  debug)
    cd lab2
    go mod download
    exec dlv test ./e2e --headless --listen=:40000 --api-version=2 --accept-multiclient -- -test.v -test.run '^TestPatientJourney$' -test.timeout=0
    ;;
  demo)
    exec bash lab2/scripts/demo.sh
    ;;
  unit)
    cd lab1
    packages=(./domain ./usecase ./storage/...)
    format=pkgname
    ;;
  integration|e2e)
    cd lab2
    packages=("./$stage")
    format=testname
    ;;
  *)
    echo "Unknown stage: $stage" >&2
    exit 2
    ;;
esac

go mod download
exec gotestsum --format "$format" \
  --junitfile "/reports/$stage/junit.xml" \
  --jsonfile "/reports/$stage/events.jsonl" \
  -- -count=1 -timeout=120s "${packages[@]}"
