#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="$ROOT_DIR/dist"

mkdir -p "$DIST_DIR/config" "$DIST_DIR/logs"
rm -f "$DIST_DIR/clinic"

cd "$ROOT_DIR"

go build -trimpath -o "$DIST_DIR/clinic" ./cmd/clinic

cp "$ROOT_DIR/config/app.json" "$DIST_DIR/config/app.json"
touch "$DIST_DIR/logs/clinic.log"

cat > "$DIST_DIR/README-demo.txt" <<'EOF'
VitaLine Clinic - demo folder

Contents:
- clinic            : single binary application
- config/app.json   : external application config
- logs/clinic.log   : log file

Quick start:
1. Start MongoDB and application services
   docker compose -f ../docker-compose.yml up -d

2. Run CLI
   ./clinic --config ./config/app.json patient list

3. Register patient
   ./clinic --config ./config/app.json patient register --full-name "Иван Иванов" --phone "+79001234567" --email "ivanov@example.com" --birth-date "2000-05-10"

4. Run API
   ./clinic api --config ./config/app.json --debug

5. Run GUI
   ./clinic gui --config ./config/app.json --debug
EOF

printf 'Components built successfully in %s\n' "$DIST_DIR"
