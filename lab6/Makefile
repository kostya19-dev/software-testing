SHELL := /bin/bash
.DEFAULT_GOAL := help

TOOL := bin/lab6
LOADTEST_REV := 7de5ab26326a1886f6bc600a563b33584ef08123
LOADTEST := .deps/$(LOADTEST_REV)
SCALE ?= 1
VARIANT ?= tracing
SCENARIO ?= stable
VARIANTS := baseline tracing logging
K6_IMAGE := grafana/k6:1.3.0
GRAFANA := docker compose -f grafana/compose.yaml

.PHONY: help setup images build run suite smoke bench compare ci check clean grafana grafana-down

help:
	@printf '%s\n' \
	  'make setup                          Собрать образы (API с OpenTelemetry, экспортер), скачать k6 и собрать утилиты' \
	  'make run VARIANT=tracing SCENARIO=stable SCALE=1' \
	  '                                    Один сценарий ЛР4 в одном варианте: baseline | tracing | logging' \
	  'make suite                          ramp, stress, stable во всех вариантах и сравнение, около 1,5 ч' \
	  'make smoke                          То же в масштабе 0.1, около 15 минут' \
	  'make bench                          Бенчмарки стоимости трассировки и логирования' \
	  'make compare                        Пересобрать results/index.html' \
	  'make ci                             Короткий прогон для CI: бенчмарки и stable во всех вариантах' \
	  'make check                          go vet и тесты lab6 и клиники' \
	  'make grafana                        Дашборд Grafana по результатам (results или evidence): http://localhost:3000' \
	  'make grafana-down                   Остановить дашборд'

setup: images build
	docker pull --quiet mongo:7
	docker pull --quiet prom/prometheus:v3.5.0
	docker pull --quiet jaegertracing/jaeger:2.11.0
	docker pull --quiet $(K6_IMAGE)

$(LOADTEST)/go.mod:
	python3 scripts/export_source.py $(LOADTEST_REV) lab4 $(LOADTEST) go.mod go.sum Dockerfile cmd internal

images: $(LOADTEST)/go.mod
	docker build --quiet --tag vitaline-lab6-api:base ../project/VitaLineClinic
	docker build --quiet --target exporter --tag vitaline-lab4-exporter:1 $(LOADTEST)

build: $(LOADTEST)/go.mod
	go -C $(LOADTEST) build -o "$(CURDIR)/bin/runner" ./cmd/runner
	go build -o $(TOOL) ./cmd/lab6

run: build
	$(TOOL) run -scale $(SCALE) -variant $(VARIANT) $(SCENARIO)

suite: build
	$(TOOL) suite -scale $(SCALE)

smoke: build
	$(TOOL) suite -scale 0.1

bench: build
	$(TOOL) bench

compare: build
	$(TOOL) compare

ci: build
	$(TOOL) bench -count 3 -benchtime 1000x
	for variant in $(VARIANTS); do $(TOOL) run -scale 0.1 -variant $$variant stable || exit 1; done
	$(TOOL) compare

check:
	go vet ./...
	go test ./...
	cd ../project/VitaLineClinic && go vet ./... && go test ./...

grafana: build
	$(TOOL) grafana
	$(GRAFANA) down --volumes --remove-orphans
	$(GRAFANA) up --detach
	$(TOOL) grafana-traces
	@echo 'Grafana: http://localhost:3000   Jaeger: http://localhost:16687'

grafana-down:
	$(GRAFANA) down --volumes --remove-orphans

clean:
	docker ps -aq --filter label=lab4.run | xargs -r docker rm -f
	docker volume ls -q | grep '^lab4-' | xargs -r docker volume rm
