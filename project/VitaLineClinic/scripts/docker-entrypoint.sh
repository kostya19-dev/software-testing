#!/usr/bin/env sh

set -eu

APP_CONFIG_PATH="${APP_CONFIG_PATH:-/app/config/app.json}"
VITALINE_APP_NAME="${VITALINE_APP_NAME:-VitaLine Clinic}"
VITALINE_ENV="${VITALINE_ENV:-docker}"
VITALINE_HTTP_ADDRESS="${VITALINE_HTTP_ADDRESS:-:8080}"
VITALINE_GUI_ADDRESS="${VITALINE_GUI_ADDRESS:-:8090}"
VITALINE_GUI_API_BASE_URL="${VITALINE_GUI_API_BASE_URL:-http://localhost:8080}"
VITALINE_DATABASE_DRIVER="${VITALINE_DATABASE_DRIVER:-mongodb}"
DATABASE_URL="${DATABASE_URL:-}"
MONGODB_DATABASE="${MONGODB_DATABASE:-vitaline_clinic}"
VITALINE_LOG_LEVEL="${VITALINE_LOG_LEVEL:-info}"
VITALINE_LOG_FILE="${VITALINE_LOG_FILE:-/app/logs/clinic.log}"
VITALINE_APPOINTMENT_MIN_LEAD_TIME_MINUTES="${VITALINE_APPOINTMENT_MIN_LEAD_TIME_MINUTES:-30}"

json_escape() {
	printf '%s' "$1" | sed 's/\\/\\\\/g; s/"/\\"/g'
}

mkdir -p "$(dirname "$APP_CONFIG_PATH")" "$(dirname "$VITALINE_LOG_FILE")"
touch "$VITALINE_LOG_FILE"

cat > "$APP_CONFIG_PATH" <<EOF
{
  "app": {
    "name": "$(json_escape "$VITALINE_APP_NAME")",
    "environment": "$(json_escape "$VITALINE_ENV")"
  },
  "api": {
    "address": "$(json_escape "$VITALINE_HTTP_ADDRESS")"
  },
  "gui": {
    "address": "$(json_escape "$VITALINE_GUI_ADDRESS")",
    "api_base_url": "$(json_escape "$VITALINE_GUI_API_BASE_URL")"
  },
  "database": {
    "driver": "$(json_escape "$VITALINE_DATABASE_DRIVER")",
    "url": "$(json_escape "$DATABASE_URL")",
    "name": "$(json_escape "$MONGODB_DATABASE")"
  },
  "logging": {
    "level": "$(json_escape "$VITALINE_LOG_LEVEL")",
    "file": "$(json_escape "$VITALINE_LOG_FILE")"
  },
  "business": {
    "appointment_min_lead_time_minutes": $VITALINE_APPOINTMENT_MIN_LEAD_TIME_MINUTES
  }
}
EOF

exec "$@" --config "$APP_CONFIG_PATH"
