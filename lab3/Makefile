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

.PHONY: allure
ifeq ($(strip $(RESULT)),)
allure: build
	$(PYTHON) scripts/run.py --trials $(TRIALS) --cpu $(CPU) --iterations $(ITERATIONS) --samples $(SAMPLES) --allure
else
allure:
	$(PYTHON) scripts/allure_run.py --result "$(RESULT)"
endif

.PHONY: hashmap
hashmap:
	go test -count=1 -v ./examples/hashmap

DEPTH ?= 3
.PHONY: nesting
nesting:
	go run ./examples/nesting -depth "$(DEPTH)"

.PHONY: build-baseline compare compare-smoke
build-baseline:
	mkdir -p .build
	CGO_ENABLED=0 go build -tags baseline -trimpath -buildvcs=false -o .build/bench-baseline ./bench

compare compare-smoke: test build build-baseline
	go test -tags baseline ./bench
	$(PYTHON) scripts/compare_depths.py --cpu $(CPU) --trials $(TRIALS) --iterations $(ITERATIONS) --samples $(SAMPLES) --allure $(if $(filter compare-smoke,$@),--smoke,)

.PHONY: formats formats-smoke
formats formats-smoke: test build
	$(PYTHON) scripts/run.py --profile formats --cpu $(CPU) --allure $(if $(filter formats-smoke,$@),--smoke --trials 2 --iterations 32 --samples 32,--trials $(TRIALS) --iterations $(ITERATIONS) --samples $(SAMPLES))
