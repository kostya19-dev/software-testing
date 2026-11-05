PYTHON ?= python3
TRIALS ?= 100
CPU ?= 0
ITERATIONS ?= 256
SAMPLES ?= 256

.PHONY: build test run smoke report
build:
	mkdir -p .build
	CGO_ENABLED=0 go build -trimpath -buildvcs=false -o .build/bench ./bench
test:
	go test ./...
	go vet ./...
	$(PYTHON) -m unittest discover -s scripts -p 'test_*.py' -v
run: test build
	$(PYTHON) scripts/run.py --trials $(TRIALS) --cpu $(CPU) --iterations $(ITERATIONS) --samples $(SAMPLES)
smoke: build
	$(PYTHON) scripts/run.py --trials 2 --cpu $(CPU) --iterations 32 --samples 32 --smoke
report:
	$(PYTHON) scripts/report.py "$(RESULT)"
