SHELL := /bin/bash
.DEFAULT_GOAL := help

RUNNER := bin/runner
SCALE ?= 1
SCENARIO ?= ramp
SCENARIOS := ramp stable stress soak soak-err10 soak-err20

.PHONY: help setup build images run suite smoke stand report index check clean $(SCENARIOS)

help:
	@printf '%s\n' \
	  'make setup                     Собрать образы и оркестратор, скачать MongoDB и Prometheus' \
	  'make ramp|stress|soak          Отдельный сценарий на свежем стенде' \
	  'make stable                    5 минут постоянной нагрузки на пределе: 400 VU, ~500 RPS' \
	  'make soak-err10|soak-err20     Длительная нагрузка с 10% / 20% ошибочных операций' \
	  'make suite                     Все сценарии по очереди' \
	  'make smoke                     Все сценарии в масштабе 0.1 для проверки' \
	  'make stand                     Поднять стенд без нагрузки, чтобы показать лимиты' \
	  'make run SCENARIO=... SCALE=...' \
	  'make report RUN=results/<id>   Пересобрать отчёт по сохранённым данным' \
	  'make index                     Обновить results/index.html' \
	  'make check                     go vet и тесты'

setup: images build
	docker pull --quiet mongo:7
	docker pull --quiet prom/prometheus:v3.5.0

images:
	docker build --quiet --tag vitaline-lab4-api:base ../project/VitaLineClinic
	docker build --quiet --target loadgen --tag vitaline-lab4-loadgen:1 .
	docker build --quiet --target exporter --tag vitaline-lab4-exporter:1 .

build:
	go build -o $(RUNNER) ./cmd/runner

run: build
	$(RUNNER) run -scale $(SCALE) $(SCENARIO)

$(SCENARIOS): build
	$(RUNNER) run -scale $(SCALE) $@

suite: build
	$(RUNNER) suite -scale $(SCALE)

stand: build
	$(RUNNER) stand

smoke: build
	$(RUNNER) suite -scale 0.1

report: build
	@test -n "$(RUN)" || { echo 'Укажите RUN=results/<id>'; exit 2; }
	$(RUNNER) report $(RUN)

index: build
	$(RUNNER) index

check:
	go vet ./...
	go test ./...

clean:
	docker ps -aq --filter label=lab4.run | xargs -r docker rm -f
	docker volume ls -q | grep '^lab4-' | xargs -r docker volume rm
