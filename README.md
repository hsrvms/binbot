# Binance Hybrid Trading Bot MVP

A decoupled, containerized trading infrastructure utilizing Go for high-frequency execution and Python for zero-allocation mathematical modeling.

## System Architecture
- **Message Broker:** NATS JetStream (At-least-once delivery, Request-Reply)
- **Ingestion/Execution:** Go 1.26 (Native Concurrency, `pgx` Ledger Management)
- **Strategy Engine:** Python 3.14 (Strictly Typed, NumPy Circular Buffers)
- **Ledger:** PostgreSQL 18.4

## 1. Environment Setup

The entire monorepo is managed via a single Devcontainer binding to a shared Docker-Compose network.

1. Ensure the Devcontainer CLI is installed globally.
2. From the repository root, build and start the environment:
```bash
devcontainer up --workspace-folder .
```
3. Drop into the interactive shell (as the `vscode` user):
```bash
devcontainer exec --workspace-folder . bash
```
4. Install all cross-language dependencies and compile Protobufs:
```bash
make setup
```

*Note: The Devcontainer image will automatically install the native Go and Python binaries, as well as the Protobuf compilers. No external SDKs are required on your host machine.*

## 2. Testing & Quality Assurance
This monorepo utilizes strict TDD. To run the full test suite across both Go and Python ecosystems:
```bash
make test
```
Note: Python tests automatically resolve imports by injecting the PYTHONPATH via the Makefile.

*A GitHub Action strictly enforces that all generated code matches the `.proto` source before any PR can be merged.*

3. Execution Commands
Start the respective engines using the Makefile commands, which automatically load configuration from the .env file.

Run the Go Ingestor:
```bash
make run-ingestor
```

Run the Python Engine:
```bash
make run-engine
```

*Configuration uses environment variables (e.g., `NATS_URL`, `BINANCE_WS_URL`) with safe local defaults.*

## 4. Validating Local Data Throughput (NATS CLI)

To manually verify that binary data is successfully streaming through JetStream without spinning up the Python engine, use the official NATS CLI tool.

**Install the NATS CLI locally within the Devcontainer:**
```bash
curl -sf https://binaries.nats.dev/nats-io/natscli/nats@latest | sh
sudo mv nats /usr/local/bin/
```

**Subscribe to the JetStream Topic:**
```bash
nats sub "market.data.BTCUSDT" --server="nats://nats:4222"
```


pip install --break-system-packages pytest nats-py

# Validating Discord Webhook Formatting
export DISCORD_WEBHOOK_URL="https://discord.com/api/webhooks/your_id/your_token"

curl -H "Content-Type: application/json" \
     -d '{"content": "🚀 **Trade Executed**\nSymbol: BTCUSDT\nPrice: 65000.50\nReason: Golden Cross: SMA10 (65000) > SMA50 (64000)"}' \
     $DISCORD_WEBHOOK_URL
