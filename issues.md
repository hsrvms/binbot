# Epic 1: Infrastructure & Core Contracts

## Issue 1: Monorepo & Devcontainer Initialization
**Dependencies / Blocked By**: None
**Context**: Establishes the unified development environment (Epic 1) to ensure the Go toolchain, Python runtime, and `protoc` compiler are isolated, reproducible, and identical across local and CI environments.
**Acceptance Criteria**:
- [ ] `docker-compose.yml` successfully spins up NATS JetStream and PostgreSQL.
- [ ] `.devcontainer` configuration successfully builds and attaches to the workspace.
- [ ] Python (with `mypy` for strict typing) and Go are globally available within the container.
- [ ] Automated tests pass (basic ping/connection checks to NATS and DB infrastructure).
**Technical Tasks & Implementation Details**:
- [ ] Create `docker-compose.yml` defining `nats:latest` (with JetStream enabled via the `-js` flag) and `postgres:15-alpine`.
- [ ] Define `.devcontainer/devcontainer.json` utilizing a base image that supports both `go` and `python3`.
- [ ] Include an initialization script in the compose file to establish the basic PostgreSQL database schema for the Ledger.

## Issue 2: Protobuf Data Contracts & CI Enforcement
**Dependencies / Blocked By**: #1
**Context**: Defines the immutable source of truth for inter-process communication between the Go ingestion layer, Python engine, and Go OMS. Prevents contract drift via containerized compilation.
**Acceptance Criteria**:
- [ ] `events.proto` encompasses `MarketTick`, `IntentSignal`, and `PortfolioState`.
- [ ] Bash script accurately compiles Go and Python bindings into their respective directories simultaneously.
- [ ] Automated tests pass (GitHub Action enforces `git diff` on the generated `/pb` folders to reject uncommitted schema updates).
**Technical Tasks & Implementation Details**:
- [ ] Create `/contracts/events.proto` strictly typing the payloads, ensuring `event_timestamp_ms` is present.
- [ ] Write `generate_protos.sh` using the `namely/protoc-all:latest` Docker image.
- [ ] Create a GitHub Actions workflow to run the script upon every push.

# Epic 2: Data Ingestion & Strategy Engine

## Issue 3: Go Ingestion - Binance WebSocket & JetStream Publisher
**Dependencies / Blocked By**: #2
**Context**: Acts as the system's "Muscle". Connects to the Binance Testnet WebSocket, normalizes the data into Protobufs, and streams it to NATS to decouple exchange turbulence from the Python engine.
**Acceptance Criteria**:
- [ ] Service connects to Binance Testnet WS and handles dropped connections natively.
- [ ] Messages are successfully published to the NATS JetStream subject `market.data.>`.
- [ ] Automated tests pass (execute a structured NATS `sub` CLI command to validate payload structure).
**Technical Tasks & Implementation Details**:
- [ ] Use native Go concurrency (`goroutines` and channels) to read from the WebSocket.
- [ ] Serialize payloads into the `MarketTick` Protobuf.
- [ ] Publish using the `nats.go` JetStream context.
- [ ] Testing Strategy: Provide a modular validation step via `nats sub market.data.BTCUSDT` to verify binary throughput on the broker.

## Issue 4: Python Engine - Pre-Allocated Ring Buffers & Signal Generation
**Dependencies / Blocked By**: #2
**Context**: Acts as the isolated "Brain" of the bot. Processes the high-frequency NATS stream using strictly typed, zero-allocation Python to calculate indicators and publish `IntentSignal` limits.
**Acceptance Criteria**:
- [ ] Code is strictly typed and passes `mypy --strict`.
- [ ] Engine subscribes to `market.data.>` and populates a fixed-size NumPy array with $O(1)$ updates.
- [ ] Publishes `IntentSignal` to NATS `strategy.intent` subject.
- [ ] Automated tests pass (assert logic execution without network dependency).
**Technical Tasks & Implementation Details**:
- [ ] Use pure Python and `numpy`. Initialize a pre-allocated array (`np.zeros(MAX_WINDOW)`) and an integer pointer for the circular buffer logic. Do NOT use `np.roll()`.
- [ ] Use `asyncio` and `nats-py` to listen to the JetStream subject via a Consumer Group.
- [ ] Serialize and publish `IntentSignal` Protobufs containing the `target_exposure` and `strategy_reasoning`.
- [ ] Testing Strategy: Use `pytest` with a mock NATS client. 

# Epic 3: OMS Execution & State Reconciliation

## Issue 5: Go OMS - Execution Loop & Transactional Ledger
**Dependencies / Blocked By**: #3, #4
**Context**: Reconciles the "Intent" (from Python) with the "Actual" Binance state. Handles Binance REST execution, partial fills, PostgreSQL ledger commits, and external webhooks.
**Acceptance Criteria**:
- [ ] Consumes `IntentSignal` from NATS.
- [ ] Executes limit orders via Binance Testnet REST API.
- [ ] Commits exact fill data to PostgreSQL in a single atomic transaction.
- [ ] Triggers Discord/Telegram webhook with the filled trade and Python's `strategy_reasoning`.
- [ ] Automated tests pass (trigger webhook manually).
**Technical Tasks & Implementation Details**:
- [ ] Parse `IntentSignal` and calculate the delta between current Postgres state and `target_exposure`.
- [ ] Implement a retry/chasing loop natively in Go for partial fills.
- [ ] Use `pgx` for PostgreSQL transactions. Ensure transactional integrity (explicit rollback if the DB write fails).
- [ ] Send JSON POST requests to webhooks natively via `net/http`.
- [ ] Testing Strategy: Provide a modular `curl` script utilizing environment variables to manually trigger and validate the webhook formatting.

## Issue 6: Go OMS - NATS State Hydration Server
**Dependencies / Blocked By**: #5
**Context**: Solves the state race-condition by running a synchronous NATS Request-Reply handler. Provides the definitive portfolio state to the Python engine immediately upon its container startup.
**Acceptance Criteria**:
- [ ] Go OMS listens on `oms.state.get`.
- [ ] Queries PostgreSQL for the latest aggregated balances.
- [ ] Returns a strictly typed `PortfolioState` Protobuf.
- [ ] Python engine synchronously blocks on this request before subscribing to the live stream.
- [ ] Automated tests pass.
**Technical Tasks & Implementation Details**:
- [ ] Implement `nats.Reply` in the Go OMS to serve the state.
- [ ] Implement `nats.Request` in the Python engine's startup sequence (`asyncio` blocking await) to ingest the state.

# Epic 4: Backtesting Pipeline

## Issue 7: Go Replayer - Lock-Step Event-Time Simulator
**Dependencies / Blocked By**: #4, #5, #6
**Context**: Replays historical Binance data through the exact same NATS infrastructure. Utilizes lock-step execution to guarantee the OMS finishes database writes before the next historical tick is sent.
**Acceptance Criteria**:
- [ ] Reads historical CSVs and streams `MarketTick` Protobufs.
- [ ] Uses NATS Request-Reply to await an "ACK" from the Python/OMS pipeline before sending the next tick.
- [ ] Absolute restriction on `time.Now()` or `datetime.now()` across the cluster during backtests.
- [ ] Automated tests pass.
**Technical Tasks & Implementation Details**:
- [ ] Create a dedicated Go worker for backtesting.
- [ ] Read CSV, serialize into `MarketTick`, mapping the CSV timestamp directly to `event_timestamp_ms`.
- [ ] Use `nats.Request` to send the tick, forcing the replayer to wait for a synchronized response from the OMS indicating the tick lifecycle is fully resolved and committed to Postgres.
- [ ] Testing Strategy: Simulate 10,000 ticks and assert the final Postgres timestamp parity against the final CSV timestamp.
