include .env
export

.PHONY: setup deps-py proto install-nats

setup: deps-py proto install-nats

install-nats:
	@echo "--- Installing NATS CLI ---"
	curl -sf https://binaries.nats.dev/nats-io/natscli/nats@latest | sh
	sudo mv nats /usr/local/bin/

deps-py:
	@echo "--- Installing Python Dependencies ---"
	pip install --break-system-packages pytest nats-py python-dotenv protobuf mypy-protobuf types-protobuf numpy

proto:
	@echo "--- Compiling Protobufs ---"
	./generate_protos.sh


.PHONY: test test-go test-py

test: test-go test-py

test-go:
	@echo "--- Running Go Tests ---"
	cd go-oms && go test ./... -v

test-py:
	@echo "--- Running Python Tests ---"
	PYTHONPATH=python-engine python -m pytest python-engine/tests/ -v

.PHONY: typecheck-py

typecheck-py:
	@echo "--- Running Mypy Strict Type Check ---"
	cd python-engine && python -m mypy .

.PHONY: run-ingestor run-engine sub-nats

run-ingestor:
	@echo "--- Starting Go Ingestion Engine ---"
	cd go-oms && go run cmd/ingestor/main.go

run-engine:
	@echo "--- Starting Python Strategy Engine ---"
	PYTHONPATH=python-engine python python-engine/main.py

sub-nats:
	@echo "--- Subscribing to NATS JetStream ---"
	nats sub "market.data.BTCUSDT" --server="nats://localhost:4222"
