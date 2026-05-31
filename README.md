# Binance Hybrid Trading Bot MVP
A decoupled, containerized trading infrastructure utilizing Go for high-frequency execution and Python for zero-allocation mathematical modeling.

## System Architecture
* **Message Broker:** NATS JetStream (At-least-once delivery, Request-Reply)

* **Ingestion/Execution:** Go 1.26 (Native Concurrency, `pgx` Ledger Management)

* **Strategy Engine:** Python 3.14 (Strictly Typed, NumPy Circular Buffers)

* **Ledger:** PostgreSQL 18.4

## 1. Environment Setup
The entire monorepo is managed via a single Devcontainer binding to a shared Docker-Compose network.

1. Ensure Docker is running locally.

2. Ensure the Devcontainer CLI is installed globally (e.g., `npm install -g @devcontainers/cli`).

3. From the repository root, build and start the environment:
```bash
devcontainer up --workspace-folder .
```

4. Drop into the interactive shell (as the `vscode` user) to run your commands:
```bash
devcontainer exec --workspace-folder . bash
```

*Note: The Devcontainer image will automatically install the native Go and Python binaries, as well as the Protobuf compilers. No external SDKs are required on your host machine.*

## 2. Compiling Protobuf Data Contracts
All inter-process communication is strictly typed via Protocol Buffers. Any time `contracts/trading/events.proto` is modified, you must regenerate the bindings natively:

```bash
./generate_protos.sh
```
A GitHub Action strictly enforces that all generated code matches the `.proto` source before any PR can be merged.

## 3. Running the Go Ingestion Engine
The Go Ingestor connects to the Binance Testnet WebSocket, normalizes JSON into Protobufs, and publishes directly to NATS JetStream.

To start the engine locally:
```bash
cd go-oms
go run cmd/ingestor/main.go
```
*Configuration uses environment variables (e.g., `NATS_URL`, `BINANCE_WS_URL`) with safe local defaults.*

## 4. Validating Local Data Throughput (NATS CLI)
To manually verify that binary data is successfully streaming through JetStream without spinning up the Python engine, use the official NATS CLI tool.

Install the NATS CLI locally within the Devcontainer:
```bash
curl -sf https://binaries.nats.dev/nats-io/natscli/nats@latest | sh
sudo mv nats /usr/local/bin/
```

Subscribe to the JetStream Topic:
```bash
nats sub "market.data.BTCUSDT" --server="nats://nats:4222"
```
```
