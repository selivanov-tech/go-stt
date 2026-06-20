.PHONY: help build server dev test lint clean

PY_STT ?= $(CURDIR)/../py-stt

help:
	@echo "make dev     - run the go service; autostarts the py-stt worker (mock engine)"
	@echo "make server  - run the go service only (expects a worker at STT_WORKER_URL)"
	@echo "make build   - build bin/go-stt"
	@echo "make test    - go test ./..."
	@echo "make lint    - golangci-lint run (install: brew install golangci-lint)"
	@echo ""
	@echo "real engines: in ../py-stt run 'make sync-mlx && make worker', then"
	@echo "  STT_DEFAULT_ENGINE=python:mlx_whisper make server"

build:
	go build -o bin/go-stt ./cmd/server

server:
	go run ./cmd/server

dev:
	STT_WORKER_AUTOSTART=true \
	STT_WORKER_CMD="$(PY_STT)/.venv/bin/uvicorn stt_worker.app:app --host 127.0.0.1 --port 8090" \
	STT_WORKER_DIR="$(PY_STT)" \
	STT_WORKER_URL=http://127.0.0.1:8090 \
	STT_DEFAULT_ENGINE=python:mock \
	go run ./cmd/server

test:
	go test ./...

lint:
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint not found — install: brew install golangci-lint"; exit 1; }
	golangci-lint run

clean:
	rm -rf bin
