# Product Requirements Document (PRD): Hybrid Binance Trading Bot MVP

## 1. Objective
To build a highly resilient, automated cryptocurrency trading bot MVP for the Binance Testnet. The system utilizes a decoupled hybrid architecture (Go + Python) communicating via NATS JetStream. The primary goal is to establish a robust infrastructure capable of handling high-frequency WebSockets and execution logic natively in Go, while isolating the strictly typed, zero-allocation Python engine for advanced indicator math and future Machine Learning integration.

## 2. Scope
### In Scope
* **Infrastructure:** A unified monorepo managed via `docker-compose` and a unified Devcontainer supporting Go, Python, and `protoc`.
* **Message Broker:** NATS JetStream deployment for at-least-once delivery, historical state retention, and synchronous Request-Reply messaging.
* **Contract Management:** Ephemeral containerized compilation of Protocol Buffers (`.proto`) to auto-generate Go structs and Python classes.
* **Ingestion (Go):** Connection to Binance Testnet WebSockets for real-time order book and price data.
* **Strategy Engine (Python):** Pure, strictly typed Python engine utilizing pre-allocated NumPy circular arrays for $O(1)$ indicator calculations. Generates declarative "Intent" signals.
* **Execution (Go OMS):** Order Management System that reconciles Python's "Intent" with the "Actual" Binance state, handling limit chasing, partial fills, and database commits.
* **Data Persistence:** PostgreSQL ledger acting as the immutable source of truth for balances, trades, and portfolio state.
* **Backtesting Pipeline:** A Go-based historical data replayer utilizing lock-step execution (NATS Request-Reply) and event-time injection to perfectly mimic live environments without ledger corruption.
* **Observability:** Go-driven outbound webhooks to Discord/Telegram for execution alerts, appending Python's strategic reasoning.

## 3. User Stories
* **As a Quant Developer**, I need the Python engine to hold pre-allocated NumPy arrays of market data, so that I can calculate technical indicators and ML predictions on every incoming tick without memory fragmentation.
* **As a Quant Developer**, I need the system to strictly enforce data contracts via Protobufs, so that type mismatches between Go and Python never cause runtime crashes.
* **As a System Architect**, I need the Go OMS to own the retry and reconciliation loop for partially filled orders, so that the Python engine remains stateless regarding network turbulence and exchange liquidity.
* **As a DevOps Engineer**, I need a single command or GitHub Action to compile the `.proto` files, so that contract drift between the Go and Python directories is impossible.
* **As a Strategy Tester**, I need the backtester to inject event-time into NATS messages and wait for the OMS to finish database commits before sending the next tick, so that my simulated ledger exactly matches my theoretical PnL.

## 4. Technical Architecture

### 4.1 Component Topology
* `/contracts`: Immutable directory containing `events.proto`.
* `/go-oms`: The Muscle. Handles Binance REST/WS, NATS publishing, state reconciliation, and Postgres writes.
* `/python-engine`: The Brain. Subscribes to NATS, hydrates state via Request-Reply, manages NumPy ring buffers, and publishes Intent limits.
* `nats-server`: NATS JetStream container.
* `postgres`: Relational database for ledger and order state.

### 4.2 Data Models (Protocol Buffers)
The system relies on strictly typed binary payloads.
```protobuf
syntax = "proto3";
package trading;

// Injected by Go Ingestion / Replayer
message MarketTick {
  string symbol = 1;
  double price = 2;
  double volume = 3;
  int64 event_timestamp_ms = 4; // Absolute source of truth for time
}

// Emitted by Python Engine
message IntentSignal {
  string symbol = 1;
  double target_exposure = 2;   // The desired final balance (e.g., 1.0 BTC)
  string strategy_reasoning = 3; // Context for alerts
  int64 signal_timestamp_ms = 4;
}

// Emitted by Go OMS on Startup Request
message PortfolioState {
  map<string, double> balances = 1;
  int64 state_timestamp_ms = 2;
}
```

### 4.3 APIs & NATS Subjects
* `market.data.>`: JetStream subject for high-frequency WebSocket ticks. Python engine subscribes using a Consumer Group to guarantee delivery.
* `strategy.intent`: Standard NATS subject for Python to send declarative exposure targets to the Go OMS.
* `oms.state.get`: Synchronous Request-Reply subject. Python queries this exactly once upon container boot to hydrate its initial portfolio state before subscribing to `market.data.>`.

### 4.4 Webhooks & Alerting
* **Trigger**: Executed exclusively by the Go OMS after a Postgres transaction commits following a successful or partially successful exchange fill.
* **Payload Construction**: Combines the actual execution data (Price, Quantity, Fees) from Binance with the `strategy_reasoning` string forwarded from the Python `IntentSignal`.
* **Target**: Discord or Telegram via standard POST requests.

## 5. Edge Cases & Error Handling
* **Python Container Restarts (Desync Prevention)**: If the Python container crashes, upon reboot it must block, execute a `nats.request` to `oms.state.get`, parse the `PortfolioState` Protobuf, hydrate its NumPy arrays, and only then bind to the live NATS stream.
* **Partial Order Fills**: If Binance partially fills an order, the Go OMS retains the `IntentSignal` in memory, updates Postgres with the partial fill, and automatically chases the remaining liquidity (e.g., via TWAP or dynamic limit updates) without querying Python. Python is only notified upon final resolution.
* **NATS JetStream Backlog**: If Python is offline for an extended period during live trading, it will read from its last acknowledged ID upon reconnecting. It will fast-forward through the queue, recalculating its NumPy arrays chronologically until it catches up to live latency.
* **Backtester Event-Time Race Conditions**: Banned use of `time.Now()` or `datetime.now()`. All database writes and system logs must be indexed using the `event_timestamp_ms` injected into the Protobuf by the historical data replayer.

## 6. Out of Scope
* **Live Mainnet Trading**: V1 is strictly restricted to the Binance Testnet for validation.
* **Complex UI/Dashboard**: All interactions will be handled via CLI, standard output, and Discord/Telegram webhooks. No React/Vue frontend will be built for V1.
* **Live Machine Learning Training Pipelines**: The Python engine will support the inference structure for future models (via NumPy), but automated model retraining pipelines are excluded from the MVP.
