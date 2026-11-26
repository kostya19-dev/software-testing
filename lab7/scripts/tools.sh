#!/usr/bin/env bash
# Устанавливает зафиксированные версии инструментов в lab7/.tools и
# экспортирует пути к ним: GOLANGCI_LINT, COMPLEXITY.
# Подключается через source из check.sh.

GOLANGCI_VERSION="${GOLANGCI_VERSION:-v2.15.0}"
TOOLS="$LAB7/.tools"
GOLANGCI_LINT="$TOOLS/golangci-lint-$GOLANGCI_VERSION"
COMPLEXITY="$TOOLS/complexity"

if ! command -v go >/dev/null 2>&1; then
	echo "go не найден в PATH" >&2
	exit 2
fi

mkdir -p "$TOOLS"
if [ ! -x "$GOLANGCI_LINT" ]; then
	echo "Установка golangci-lint $GOLANGCI_VERSION в $TOOLS"
	GOBIN="$TOOLS" GOFLAGS=-mod=mod go install "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$GOLANGCI_VERSION"
	mv "$TOOLS/golangci-lint" "$GOLANGCI_LINT"
fi

# Анализатор Холстеда собирается из исходников lab7 при каждом запуске:
# go кэширует сборку, а изменения в lab7/internal/metrics сразу учитываются.
(cd "$LAB7" && go build -o "$COMPLEXITY" ./cmd/complexity)

export GOLANGCI_LINT COMPLEXITY
