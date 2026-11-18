SHELL := /bin/bash
.DEFAULT_GOAL := test
.PHONY: test check
test:
	python3 scripts/run.py
check:
	go vet ./...
	go test ./...
