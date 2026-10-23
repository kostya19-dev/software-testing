SHELL := /bin/bash
.DEFAULT_GOAL := help

ifneq ($(wildcard /snap/go/current/bin/go),)
export PATH := /snap/go/current/bin:$(PATH)
endif

CASE ?=
SUITE ?= all
SUITES ?=
SEED ?= off
PARALLEL_PACKAGES ?= 1
RESULTS ?= reports/normal
PORT ?= 8081
ALLURE := .tools/allure-cli/node_modules/.bin/allure
export SUITE SUITES SEED PARALLEL_PACKAGES CASE
export PYTHONDONTWRITEBYTECODE := 1

.PHONY: help setup test test-domain test-usecase test-memory test-mongodb \
        errors case check-docs shuffle offline race coverage report show clean all

help:
	@printf '%s\n' \
	  'make setup       Установить инструменты' \
	  'make test        Запустить тесты и собрать HTML-отчёт' \
	  'make test-domain / test-usecase / test-memory / test-mongodb' \
	  'make errors      Проверить исключительные ситуации' \
	  'make case CASE=TC-06-04  Запустить строку из документации' \
	  'make check-docs  Проверить ссылки на тесты по полному прогону' \
	  'make shuffle     Перемешать тесты (SEED=42 для повтора)' \
	  'make offline     Запустить тесты без сети' \
	  'make race        Проверить гонки данных' \
	  'make coverage    Измерить покрытие операторов и ветвей' \
	  'make report      Собрать Allure из RESULTS=reports/normal' \
	  'make show        Показать Allure: PORT=8081 или свободный порт' \
	  'make clean       Удалить результаты запусков' \
	  'make all         Тесты, покрытие, shuffle, offline и Allure'

setup:
	bash scripts/setup.sh

test:
	bash scripts/run.sh normal

test-domain test-usecase test-memory test-mongodb:
	bash scripts/run.sh normal $(@:test-%=%)

errors:
	TEST_PATTERN='Errors|Transitions|Rejection' bash scripts/run.sh errors

case:
	@test -n "$$CASE" || { echo "Укажите CASE=TC-06-04"; exit 2; }
	python3 scripts/cases.py run "$$CASE"

check-docs:
	python3 scripts/cases.py check reports/normal/junit.xml

shuffle:
	bash scripts/run.sh shuffle

offline:
	bash scripts/offline.sh

race:
	go test -race -count=1 ./domain ./usecase ./storage/...

coverage:
	bash scripts/coverage.sh

report:
	@test -x "$(ALLURE)" || { echo 'Сначала выполните make setup'; exit 2; }
	@test -f "$(RESULTS)/junit.xml" || { echo 'Нет результатов: сначала выполните make test'; exit 2; }
	python3 scripts/report.py "$(RESULTS)"

show: report
	python3 scripts/show.py "$(RESULTS)/allure" --port "$(PORT)"

clean:
	python3 -c 'from pathlib import Path; import shutil; p = Path("reports"); assert not p.is_symlink(); shutil.rmtree(p, ignore_errors=True)'

all:
	$(MAKE) test
	$(MAKE) check-docs
	$(MAKE) coverage
	$(MAKE) shuffle
	$(MAKE) offline
