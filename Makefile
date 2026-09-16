.PHONY: help build server dev up down restart status logs test lint clean

PY_STT      ?= $(CURDIR)/../py-stt
ADDR        ?= 127.0.0.1:8080
WORKER_PORT ?= 8090
DEV_ENGINE  ?= python:mock
UP_ENGINE   ?= python:mlx_whisper
RUN_DIR     := $(CURDIR)/var
PID_FILE    := $(RUN_DIR)/go-stt.pid
LOG_FILE    := $(RUN_DIR)/go-stt.log

# The supervisor splits STT_WORKER_CMD on spaces, so PY_STT must not contain any.
WORKER_ENV = STT_ADDR=$(ADDR) \
	STT_WORKER_AUTOSTART=true \
	STT_WORKER_CMD="$(PY_STT)/.venv/bin/python -m uvicorn stt_worker.app:app --host 127.0.0.1 --port $(WORKER_PORT)" \
	STT_WORKER_DIR="$(PY_STT)" \
	STT_WORKER_URL=http://127.0.0.1:$(WORKER_PORT)

help:
	@echo "make dev     - run in the foreground; autostarts the py-stt worker (mock engine)"
	@echo "make up      - run in the background with real local models (default $(UP_ENGINE))"
	@echo "make down    - stop the background service and its worker"
	@echo "make status  - health and the engine list with install status"
	@echo "make logs    - follow the service and worker log"
	@echo "make server  - run the go service only (expects a worker at STT_WORKER_URL)"
	@echo "make build   - build bin/go-stt"
	@echo "make test    - go test ./..."
	@echo "make lint    - golangci-lint run (install: brew install golangci-lint)"
	@echo ""
	@echo "local models: in ../py-stt run 'make sync-local' once (MLX Whisper + Parakeet)"

build:
	go build -o bin/go-stt ./cmd/server

server:
	go run ./cmd/server

dev:
	$(WORKER_ENV) STT_DEFAULT_ENGINE=$(DEV_ENGINE) go run ./cmd/server

up: build
	@mkdir -p "$(RUN_DIR)"; \
	if [ -f "$(PID_FILE)" ] && kill -0 "$$(cat "$(PID_FILE)")" 2>/dev/null; then \
		echo "go-stt already running (pid $$(cat "$(PID_FILE)"))"; exit 0; \
	fi; \
	$(WORKER_ENV) STT_DEFAULT_ENGINE=$(UP_ENGINE) nohup bin/go-stt >> "$(LOG_FILE)" 2>&1 & \
	echo $$! > "$(PID_FILE)"; \
	for i in $$(seq 1 90); do curl -fsS "http://$(ADDR)/healthz" >/dev/null 2>&1 && break; sleep 1; done; \
	curl -fsS "http://$(ADDR)/healthz" >/dev/null 2>&1 || { echo "go-stt did not come up, see $(LOG_FILE)"; exit 1; }; \
	echo "go-stt up on http://$(ADDR) (pid $$(cat "$(PID_FILE)"), log $(LOG_FILE))"

down:
	@if [ -f "$(PID_FILE)" ] && kill -0 "$$(cat "$(PID_FILE)")" 2>/dev/null; then \
		kill "$$(cat "$(PID_FILE)")"; \
		for i in $$(seq 1 20); do kill -0 "$$(cat "$(PID_FILE)")" 2>/dev/null || break; sleep 1; done; \
		echo "go-stt stopped"; \
	else \
		echo "go-stt is not running"; \
	fi; \
	rm -f "$(PID_FILE)"

restart: down up

status:
	@curl -fsS "http://$(ADDR)/healthz" && curl -fsS "http://$(ADDR)/v1/engines" \
		|| echo "go-stt is not reachable on $(ADDR)"

logs:
	tail -n 100 -f "$(LOG_FILE)"

test:
	go test ./...

lint:
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint not found — install: brew install golangci-lint"; exit 1; }
	golangci-lint run

clean:
	rm -rf bin
