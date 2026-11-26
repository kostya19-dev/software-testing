SHELL := /bin/bash
.DEFAULT_GOAL := help

APP := ../project/VitaLineClinic
UNIT_TEST_PACKAGES := ./cmd/clinic ./internal/appconfig ./internal/bootstrap ./internal/delivery/cli \
	./internal/delivery/http ./internal/delivery/gui ./internal/domain ./internal/storage/memory \
	./internal/usecase ./internal/logging

.PHONY: help check lint complexity test ci install-hooks uninstall-hooks demo-hook clean

help:
	@printf '%s\n' \
	  'make check             Статический анализ: golangci-lint (стиль, типы, gocyclo<=10) и метрики Холстеда' \
	  'make complexity        Только отчёт по цикломатической сложности и метрикам Холстеда' \
	  'make test              Тесты анализатора и юнит-тесты клиники' \
	  'make ci                То же, что в GitHub Actions: check, затем test' \
	  'make install-hooks     Включить pre-commit хук (git config core.hooksPath .githooks)' \
	  'make uninstall-hooks   Выключить хук' \
	  'make demo-hook         Показать, что хук блокирует коммит с CC > 10, а --no-verify пропускает'

check lint:
	scripts/check.sh

complexity:
	@LAB7=$(CURDIR); source scripts/tools.sh && "$$COMPLEXITY" -dir $(APP) -top 20

test:
	go test ./...
	cd $(APP) && go test $(UNIT_TEST_PACKAGES)

ci: check test

install-hooks:
	git config core.hooksPath .githooks
	@echo "pre-commit включён: $$(git rev-parse --show-toplevel)/.githooks/pre-commit"

uninstall-hooks:
	git config --unset core.hooksPath || true

demo-hook:
	scripts/demo-hook.sh

clean:
	rm -rf .tools reports
