#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
goroot=$(go env GOROOT)
export PATH="$goroot/bin:$PATH"
exec unshare --user --map-root-user --net env GOPROXY=off GOSUMDB=off GOTOOLCHAIN=local bash scripts/run.sh offline
