#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
if [[ ! -f ci/.env ]]; then
  umask 077
  printf 'LAB_ROOT=%s\nLAB_ADMIN_PASSWORD=%s\nJENKINS_PORT=8092\n' "$(cd .. && pwd)" "$(python3 -c 'import secrets; print(secrets.token_urlsafe(24))')" > ci/.env
fi
set -a
source ci/.env
set +a
compose=(docker compose -p tiopo-lab2-ci --env-file ci/.env -f ci/compose.yaml)
case "${1:-up}" in
 up)
   docker image inspect tiopo-lab2-runner:2 >/dev/null 2>&1 || make setup
   "${compose[@]}" up -d --build
   ready=0
   for attempt in {1..120}; do
     if curl -fsS "http://localhost:$JENKINS_PORT/login" >/dev/null 2>&1; then ready=1; break; fi
     sleep 1
   done
   [[ "$ready" == 1 ]] || { echo 'Jenkins ещё не готов; смотрите docker compose logs' >&2; exit 1; }
   echo "Jenkins: http://localhost:$JENKINS_PORT · логин student · пароль в ci/.env"
   ;;
 down) "${compose[@]}" down ;;
 build) python3 scripts/jenkins_build.py ;;
 *) echo 'Use up, down or build' >&2; exit 2 ;;
esac
