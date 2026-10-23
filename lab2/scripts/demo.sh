#!/usr/bin/env bash
set -euo pipefail
mkdir -p /reports/traffic
exec > >(tee /reports/traffic/requests.txt) 2>&1
base=${API_URL:?}
[[ "$base" == "http://api-${RUN_ID:?}:8080" ]]
request() {
  local expected=$1 method=$2 path=$3 data=${4:-}
  local args=(-sS --max-time 15 -X "$method" -H 'Content-Type: application/json' -o /tmp/response.json -w '%{http_code}')
  [[ -z "$data" ]] || args+=(-d "$data")
  local code
  code=$(curl "${args[@]}" "$base$path")
  echo "$method $path -> HTTP $code" >&2
  cat /tmp/response.json >&2
  [[ "$code" == "$expected" ]]
  cat /tmp/response.json
}
email="$(cat /proc/sys/kernel/random/uuid)@example.test"
patient=$(request 201 POST /api/patients "{\"full_name\":\"Пациент демонстрации\",\"phone\":\"+79001234567\",\"email\":\"$email\",\"birth_date\":\"1990-01-01T00:00:00Z\"}")
pid=$(jq -er .ID <<< "$patient")
doctor=$(cat /proc/sys/kernel/random/uuid)
date=$(date -u -d 'tomorrow 10:00' +%Y-%m-%dT%H:%M:%SZ)
input=$(jq -nc --arg p "$pid" --arg d "$doctor" --arg date "$date" '{patient_id:$p,doctor_id:$d,date:$date}')
app=$(request 201 POST /api/appointments "$input")
aid=$(jq -er .ID <<< "$app")
request 409 POST /api/appointments "$input" >/dev/null
visit=$(request 201 POST /api/visits "{\"appointment_id\":\"$aid\",\"complaints\":\"Головная боль\"}")
vid=$(jq -er .ID <<< "$visit")
request 200 POST "/api/visits/$vid/complete" '{"diagnosis":"Тестовый диагноз","recommendations":"Контрольный визит"}' >/dev/null
history=$(request 200 GET "/api/visits?patient_id=$pid")
jq -e --arg id "$vid" 'length == 1 and .[0].ID == $id and .[0].Status == "COMPLETED" and .[0].Diagnosis == "Тестовый диагноз"' <<< "$history" >/dev/null
bookings=$(request 200 GET "/api/appointments?patient_id=$pid")
jq -e --arg id "$aid" 'length == 1 and .[0].ID == $id and .[0].Status == "COMPLETED"' <<< "$bookings" >/dev/null
echo 'Demo scenario passed; inspect demo.pcap in Wireshark.'
