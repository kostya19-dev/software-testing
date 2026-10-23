SHELL := /bin/bash
.DEFAULT_GOAL := help

RUN_ID ?=
export RUN_ID
COUNT ?= 5
export COUNT
export PYTHONDONTWRITEBYTECODE := 1

.PHONY: help setup up down status test repeat parallel parallel-logs check-interruption recover debug debug-connect ci ci-stop ci-run

help:
	@printf '%s\n' \
	  'make setup               Собрать образ с Go, Allure и Delve' \
	  'make up                  Поднять постоянный стенд' \
	  'make status              Состояние MongoDB, API и сервера отчётов' \
	  'make test                Unit → integration → E2E; отчёт Allure' \
	  'make repeat              Два последовательных прогона' \
	  'make parallel            Пять прогонов на уже поднятом стенде' \
	  'make parallel-logs       Логи последнего параллельного запуска' \
	  'make check-interruption  Проверить откат после SIGTERM' \
	  'make debug RUN_ID=...    Запустить e2e под Delve' \
	  'make recover RUN_ID=...  Очистить только указанную сессию' \
	  'make down                Остановить постоянный стенд' \
	  'make ci / ci-run / ci-stop  Управление Jenkins'

setup:
	docker build -t tiopo-lab2-runner:2 -f docker/runner.Dockerfile .

up down status:
	python3 scripts/stand.py $@

test:
	python3 scripts/pipeline.py all

repeat:
	$(MAKE) test RUN_ID=
	$(MAKE) test RUN_ID=

parallel:
	@bash scripts/parallel.sh

parallel-logs:
	@python3 scripts/parallel_logs.py

check-interruption:
	python3 scripts/check-interruption.py

recover:
	@test -n "$$RUN_ID" || { echo 'Укажите RUN_ID из имени отчёта'; exit 2; }
	python3 scripts/pipeline.py finish

debug:
	@test -n "$$RUN_ID" || { echo 'Укажите новое имя RUN_ID или сохранённую сессию'; exit 2; }
	python3 scripts/pipeline.py debug

debug-connect:
	@test -n "$$RUN_ID" || { echo "Укажите RUN_ID сессии отладки"; exit 2; }
	python3 scripts/pipeline.py connect

ci:
	bash scripts/jenkins.sh up

ci-stop:
	bash scripts/jenkins.sh down

ci-run:
	bash scripts/jenkins.sh build
