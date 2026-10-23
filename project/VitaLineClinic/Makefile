SHELL := /bin/bash
.DEFAULT_GOAL := help

APP ?= ./dist/clinic
CONFIG ?= ./config/app.json
DOCKER ?= docker
COMPOSE ?= $(DOCKER) compose -f ../docker-compose.yml --project-directory ..
DB_HOST ?= localhost
DB_PORT ?= 27017
DB_NAME ?= vitaline_clinic
DATABASE_URL ?= mongodb://localhost:27017

DOCTOR_ID ?= aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa
PATIENT_ID ?= 22222222-2222-2222-2222-222222222222
APPOINTMENT_ID ?= 44444444-4444-4444-4444-444444444444
VISIT_ID ?= 77777777-7777-7777-7777-777777777777
SLOT_ID ?= 55555555-5555-5555-5555-555555555555

FULL_NAME ?= Ivan Ivanov
PHONE ?= +79001234567
EMAIL ?= ivanov@example.com
BIRTH_DATE ?= 1990-05-15

DATE ?= 2030-05-10
START ?= 09:00
END ?= 09:30
CABINET ?= 101
DATE_TIME ?= 2030-05-10 09:00

COMPLAINTS ?= Headache
DIAGNOSIS ?= ARVI
RECOMMENDATIONS ?= Bed rest

ARGS ?= patient list
GO_TEST_FLAGS ?=
UNIT_TEST_PACKAGES ?= ./cmd/clinic ./internal/appconfig ./internal/bootstrap ./internal/delivery/cli ./internal/delivery/http ./internal/delivery/gui ./internal/domain ./internal/storage/memory ./internal/usecase ./internal/logging
GUI_URL ?= http://localhost:8090

.PHONY: help build demo \
	docker-up docker-down docker-ps docker-logs docker-gui-logs api-health api-info gui-status gui-run \
	db-seed db-seed-local db-seed-docker db-counts \
	test test-verbose test-postgres-connection test-integration \
	cli \
	patient-list patient-register patient-get patient-search patient-delete \
	schedule-create schedule-list schedule-free schedule-delete \
	appointment-create appointment-cancel appointment-get appointment-list appointment-list-patient \
	visit-start visit-complete visit-get-appointment visit-list-patient \
	logs logs-follow

help:
	@printf '\nVitaLine Clinic - lab 9 menu\n'
	@printf 'Usage: make <target> [VARIABLE=value]\n\n'
	@printf 'Build and runtime:\n'
	@printf '  make build                         build dist/clinic single binary\n'
	@printf '  make test                          run safe Go tests without DB truncation\n'
	@printf '  make test-verbose                  run safe Go tests with verbose output\n'
	@printf '  make test-postgres-connection      run PostgreSQL connection unit tests\n'
	@printf '  make test-integration              run PostgreSQL integration tests and reseed DB\n'
	@printf '  make docker-up                     start MongoDB, migrations, backend and frontend\n'
	@printf '                                      use DOCKER="sudo docker" if Docker socket needs sudo\n'
	@printf '  make docker-down                   stop Docker Compose services\n'
	@printf '  make docker-ps                     show containers\n'
	@printf '  make api-health                    GET /health\n'
	@printf '  make api-info                      GET /info\n\n'
	@printf '  make gui-status                    check GUI HTTP page\n'
	@printf '  make gui-run                       run GUI mode locally on configured address\n\n'
	@printf 'Database:\n'
	@printf '  make db-seed                       apply MongoDB migration/seed script to localhost DB\n'
	@printf '  make db-seed-docker                apply MongoDB migration/seed script inside Docker DB\n'
	@printf '  make db-counts                     show row counts\n\n'
	@printf 'Generic CLI:\n'
	@printf '  make cli ARGS="patient list"\n\n'
	@printf 'Patient commands:\n'
	@printf '  make patient-list\n'
	@printf '  make patient-register FULL_NAME="Ivan Ivanov" PHONE="+79001234567" EMAIL="ivan@example.com" BIRTH_DATE=1990-05-15\n'
	@printf '  make patient-get PATIENT_ID=<uuid>\n'
	@printf '  make patient-search EMAIL=ivanov@example.com\n'
	@printf '  make patient-delete PATIENT_ID=<uuid>\n\n'
	@printf 'Schedule commands:\n'
	@printf '  make schedule-create DOCTOR_ID=<uuid> DATE=2030-05-10 START=09:00 END=09:30 CABINET=101\n'
	@printf '  make schedule-list DOCTOR_ID=<uuid>\n'
	@printf '  make schedule-free DOCTOR_ID=<uuid> DATE=2030-05-10\n'
	@printf '  make schedule-delete SLOT_ID=<uuid>\n\n'
	@printf 'Appointment commands:\n'
	@printf '  make appointment-create PATIENT_ID=<uuid> DOCTOR_ID=<uuid> DATE_TIME="2030-05-10 09:00"\n'
	@printf '  make appointment-list\n'
	@printf '  make appointment-list-patient PATIENT_ID=<uuid>\n'
	@printf '  make appointment-get APPOINTMENT_ID=<uuid>\n'
	@printf '  make appointment-cancel APPOINTMENT_ID=<uuid>\n\n'
	@printf 'Visit commands:\n'
	@printf '  make visit-start APPOINTMENT_ID=<uuid> COMPLAINTS="Headache"\n'
	@printf '  make visit-complete VISIT_ID=<uuid> DIAGNOSIS="ARVI" RECOMMENDATIONS="Bed rest"\n'
	@printf '  make visit-get-appointment APPOINTMENT_ID=<uuid>\n'
	@printf '  make visit-list-patient PATIENT_ID=<uuid>\n\n'
	@printf 'Logs:\n'
	@printf '  make logs                          show last application log lines\n'
	@printf '  make logs-follow                   follow application log file\n\n'

build:
	@bash scripts/build_components.sh

test:
	@go test $(GO_TEST_FLAGS) $(UNIT_TEST_PACKAGES)

test-verbose:
	@$(MAKE) test GO_TEST_FLAGS="-v"

test-postgres-connection:
	@go test $(GO_TEST_FLAGS) ./internal/storage/postgres -run 'TestOpen'

test-integration:
	@printf 'Running storage integration tests. PostgreSQL tests run only when TEST_DATABASE_URL or DATABASE_URL points to PostgreSQL.\n'
	@go test $(GO_TEST_FLAGS) ./internal/storage/postgres

docker-up:
	@$(COMPOSE) up -d --build

docker-down:
	@$(COMPOSE) down

docker-ps:
	@$(COMPOSE) ps

docker-logs:
	@$(COMPOSE) logs -f api

docker-gui-logs:
	@$(COMPOSE) logs -f frontend

api-health:
	@curl -s http://localhost:8080/health
	@printf '\n'

api-info:
	@curl -s http://localhost:8080/info
	@printf '\n'

gui-status:
	@curl -s -o /dev/null -w "GUI status: %{http_code} $(GUI_URL)/patients\n" "$(GUI_URL)/patients"

gui-run:
	@$(APP) gui --config $(CONFIG)

db-seed: db-seed-local

db-seed-local:
	@MONGODB_DATABASE="$(DB_NAME)" mongosh "$(DATABASE_URL)/$(DB_NAME)" migrations/mongo/001-init.js

db-seed-docker:
	@$(COMPOSE) up --force-recreate migrations

db-counts:
	@$(DOCKER) exec vitaline_mongo mongosh "$(DB_NAME)" --quiet --eval 'printjson({patients: db.patients.countDocuments(), appointments: db.appointments.countDocuments(), slots: db.schedule_slots.countDocuments(), visits: db.visits.countDocuments(), emergency_requests: db.emergency_requests.countDocuments(), reschedule_offers: db.reschedule_offers.countDocuments()})'

cli:
	@$(APP) --config $(CONFIG) $(ARGS)

demo: patient-list appointment-list schedule-list visit-list-patient

patient-list:
	@$(APP) --config $(CONFIG) patient list

patient-register:
	@$(APP) --config $(CONFIG) patient register --full-name "$(FULL_NAME)" --phone "$(PHONE)" --email "$(EMAIL)" --birth-date "$(BIRTH_DATE)"

patient-get:
	@$(APP) --config $(CONFIG) patient get --id "$(PATIENT_ID)"

patient-search:
	@$(APP) --config $(CONFIG) patient search --email "$(EMAIL)"

patient-delete:
	@$(APP) --config $(CONFIG) patient delete --id "$(PATIENT_ID)"

schedule-create:
	@$(APP) --config $(CONFIG) schedule create --doctor-id "$(DOCTOR_ID)" --date "$(DATE)" --start "$(START)" --end "$(END)" --cabinet "$(CABINET)"

schedule-list:
	@$(APP) --config $(CONFIG) schedule list --doctor-id "$(DOCTOR_ID)"

schedule-free:
	@$(APP) --config $(CONFIG) schedule free --doctor-id "$(DOCTOR_ID)" --date "$(DATE)"

schedule-delete:
	@$(APP) --config $(CONFIG) schedule delete --id "$(SLOT_ID)"

appointment-create:
	@$(APP) --config $(CONFIG) appointment create --patient-id "$(PATIENT_ID)" --doctor-id "$(DOCTOR_ID)" --date-time "$(DATE_TIME)"

appointment-cancel:
	@$(APP) --config $(CONFIG) appointment cancel --id "$(APPOINTMENT_ID)"

appointment-get:
	@$(APP) --config $(CONFIG) appointment get --id "$(APPOINTMENT_ID)"

appointment-list:
	@$(APP) --config $(CONFIG) appointment list

appointment-list-patient:
	@$(APP) --config $(CONFIG) appointment list-patient --patient-id "$(PATIENT_ID)"

visit-start:
	@$(APP) --config $(CONFIG) visit start --appointment-id "$(APPOINTMENT_ID)" --complaints "$(COMPLAINTS)"

visit-complete:
	@$(APP) --config $(CONFIG) visit complete --visit-id "$(VISIT_ID)" --diagnosis "$(DIAGNOSIS)" --recommendations "$(RECOMMENDATIONS)"

visit-get-appointment:
	@$(APP) --config $(CONFIG) visit get-appointment --appointment-id "$(APPOINTMENT_ID)"

visit-list-patient:
	@$(APP) --config $(CONFIG) visit list-patient --patient-id "$(PATIENT_ID)"

logs:
	@tail -n 30 logs/clinic.log

logs-follow:
	@tail -f logs/clinic.log
