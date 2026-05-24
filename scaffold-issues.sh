#!/usr/bin/env bash
set -e

# --- Configuration & Validation ---
if ! command -v gh &> /dev/null; then
    echo "Error: GitHub CLI (gh) is not installed. Please install and run 'gh auth login'."
    exit 1
fi

REPO=$(gh repo view --json nameWithOwner -q .nameWithOwner 2>/dev/null)
if [ -z "$REPO" ]; then
    echo "Error: You are not in a valid GitHub repository directory, or you are not authenticated."
    exit 1
fi

echo "🚀 Bootstrapping Binance Bot MVP Issues & Milestones for $REPO..."

# --- 1. Label Generation ---
echo "🏷️  Creating taxonomy labels..."
gh label create "type: feature" -c "0E8A16" -d "New feature development" --force || true
gh label create "type: tech-debt" -c "FBCA04" -d "Technical debt" --force || true
gh label create "status: blocked" -c "B60205" -d "Blocked by another issue" --force || true
gh label create "component: infra" -c "1D76DB" -d "Docker, CI/CD, Contracts" --force || true
gh label create "component: go-oms" -c "00ADD8" -d "Go Execution Layer" --force || true
gh label create "component: python-engine" -c "3572A5" -d "Python Strategy Engine" --force || true
gh label create "component: nats" -c "27AB83" -d "NATS JetStream IPC" --force || true

# --- 2. Milestone Generation ---
echo "🗺️  Creating or verifying Epics as Milestones..."
create_milestone() {
  local TITLE=$1
  # Create milestone via API; if it already exists, gracefully ignore the error
  gh api repos/"$REPO"/milestones -f title="$TITLE" >/dev/null 2>&1 || true
  # gh issue create expects the milestone TITLE, not the ID number
  echo "$TITLE"
}

MS_1=$(create_milestone "Epic 1: Infrastructure & Core Contracts")
MS_2=$(create_milestone "Epic 2: Data Ingestion & Strategy Engine")
MS_3=$(create_milestone "Epic 3: OMS Execution & State Reconciliation")
MS_4=$(create_milestone "Epic 4: Backtesting Pipeline")

# --- 3. Issue Generation ---
echo "🎫 Creating Issues..."

create_issue() {
  local TITLE=$1
  local MILESTONE=$2
  local LABELS=$3
  local DEPENDENCIES=$4
  local CONTEXT=$5
  local ACCEPTANCE=$6
  local TECHNICAL=$7

  gh issue create --title "$TITLE" --milestone "$MILESTONE" --label "$LABELS" --body-file - <<EOF
**Development Task**
Ensure this task is modular, strictly typed, and includes a clear testing strategy.

### Dependencies / Blocked By
$DEPENDENCIES

### Context
$CONTEXT

### Acceptance Criteria
$ACCEPTANCE

### Technical Tasks & Implementation Details
$TECHNICAL
EOF
  echo "Created: $TITLE"
  sleep 1 # Prevent hitting GitHub API rate limits
}

# --- Issue 1 ---
create_issue \
  "Monorepo & Devcontainer Initialization" \
  "$MS_1" \
  "component: infra,type: feature" \
  "None" \
  "Establishes the unified development environment to ensure the toolchain is isolated, reproducible, and identical across local and CI environments." \
  "- [ ] \`docker-compose.yml\` successfully spins up NATS JetStream and PostgreSQL.
- [ ] \`.devcontainer\` configuration successfully builds and attaches to the workspace.
- [ ] Python (with \`mypy\`) and Go are globally available within the container.
- [ ] Automated tests pass (basic ping/connection checks to NATS and DB infrastructure)." \
  "- [ ] Create \`docker-compose.yml\` defining \`nats:latest\` (with JetStream flag) and \`postgres:16-alpine\`.
- [ ] Define \`.devcontainer/devcontainer.json\` utilizing a base image natively supporting Go 1.24 and Python 3.13.
- [ ] Include an initialization script to establish the basic PostgreSQL DB schema for the Ledger."

# --- Issue 2 ---
create_issue \
  "Protobuf Data Contracts & CI Enforcement" \
  "$MS_1" \
  "component: infra,type: feature,status: blocked" \
  "Depends on #1" \
  "Defines the immutable source of truth for inter-process communication between Go and Python. Prevents contract drift via containerized compilation." \
  "- [ ] \`events.proto\` encompasses \`MarketTick\`, \`IntentSignal\`, and \`PortfolioState\`.
- [ ] Bash script accurately compiles Go and Python bindings simultaneously.
- [ ] Automated tests pass (GitHub Action enforces \`git diff\` on generated folders)." \
  "- [ ] Create \`/contracts/events.proto\` strictly typing payloads (must include \`event_timestamp_ms\`).
- [ ] Write \`generate_protos.sh\` using the \`namely/protoc-all:latest\` Docker image.
- [ ] Create a GitHub Actions workflow to run the script upon every push to main."

# --- Issue 3 ---
create_issue \
  "Go Ingestion - Binance WebSocket & JetStream Publisher" \
  "$MS_2" \
  "component: go-oms,component: nats,type: feature,status: blocked" \
  "Depends on #2" \
  "Acts as the system's Muscle. Connects to the Binance Testnet WebSocket, normalizes data into Protobufs, and streams it to NATS to decouple exchange turbulence from the Python engine." \
  "- [ ] Service connects to Binance Testnet WS and handles dropped connections natively.
- [ ] Messages are successfully published to NATS subject \`market.data.>\`.
- [ ] Automated tests pass (execute a structured NATS \`sub\` CLI command to validate payload structure)." \
  "- [ ] Use native Go 1.24 concurrency (\`goroutines\`/channels) to read the WebSocket.
- [ ] Serialize payloads into the \`MarketTick\` Protobuf.
- [ ] Publish using the \`nats.go\` JetStream context.
- [ ] Testing Strategy: Provide a modular validation step via \`nats sub market.data.BTCUSDT\`."

# --- Issue 4 ---
create_issue \
  "Python Engine - Pre-Allocated Ring Buffers & Signal Generation" \
  "$MS_2" \
  "component: python-engine,type: feature,status: blocked" \
  "Depends on #2" \
  "Acts as the isolated Brain. Processes the high-frequency stream using strictly typed, zero-allocation Python 3.13 to calculate indicators and publish IntentSignals." \
  "- [ ] Code is strictly typed and passes \`mypy --strict\`.
- [ ] Engine subscribes to \`market.data.>\` and populates a fixed-size NumPy array with O(1) updates.
- [ ] Publishes \`IntentSignal\` to NATS \`strategy.intent\` subject.
- [ ] Automated tests pass (assert logic execution without network dependency)." \
  "- [ ] Use pure Python and \`numpy\`. Initialize a pre-allocated array (\`np.zeros(MAX_WINDOW)\`) and integer pointer for circular logic. Do NOT use \`np.roll()\`.
- [ ] Use \`asyncio\` and \`nats-py\` to listen via a JetStream Consumer Group.
- [ ] Serialize and publish \`IntentSignal\` Protobufs containing \`target_exposure\`.
- [ ] Testing Strategy: Use \`pytest\` with a mock NATS client."

# --- Issue 5 ---
create_issue \
  "Go OMS - Execution Loop & Transactional Ledger" \
  "$MS_3" \
  "component: go-oms,type: feature,status: blocked" \
  "Depends on #3, #4" \
  "Reconciles Intent (from Python) with Actual Binance state. Handles REST execution, partial fills, PostgreSQL ledger commits, and external webhooks." \
  "- [ ] Consumes \`IntentSignal\` from NATS.
- [ ] Executes limit orders via Binance Testnet REST API.
- [ ] Commits exact fill data to PostgreSQL atomically.
- [ ] Triggers Discord/Telegram webhook with filled trade and \`strategy_reasoning\`.
- [ ] Automated tests pass." \
  "- [ ] Parse \`IntentSignal\` and calculate delta between Postgres state and \`target_exposure\`.
- [ ] Implement a retry/chasing loop natively in Go for partial fills.
- [ ] Use \`pgx\` for PostgreSQL transactions with explicit rollbacks if DB writes fail.
- [ ] Send JSON POST requests to webhooks natively via \`net/http\`.
- [ ] Testing Strategy: Provide a modular \`curl\` script for manual webhook formatting validation."

# --- Issue 6 ---
create_issue \
  "Go OMS - NATS State Hydration Server" \
  "$MS_3" \
  "component: go-oms,component: python-engine,type: feature,status: blocked" \
  "Depends on #5" \
  "Solves the state race-condition via synchronous NATS Request-Reply. Provides definitive portfolio state to Python immediately upon startup." \
  "- [ ] Go OMS listens on \`oms.state.get\`.
- [ ] Queries PostgreSQL for the latest aggregated balances.
- [ ] Returns a strictly typed \`PortfolioState\` Protobuf.
- [ ] Python engine synchronously blocks on this request before live subscription.
- [ ] Automated tests pass." \
  "- [ ] Implement \`nats.Reply\` in Go OMS to serve the state.
- [ ] Implement \`nats.Request\` in the Python engine's startup sequence (\`asyncio\` await) to ingest state."

# --- Issue 7 ---
create_issue \
  "Go Replayer - Lock-Step Event-Time Simulator" \
  "$MS_4" \
  "component: go-oms,component: infra,type: feature,status: blocked" \
  "Depends on #4, #5, #6" \
  "Replays historical data through NATS using lock-step execution to guarantee OMS finishes database writes before the next historical tick is sent." \
  "- [ ] Reads historical CSVs and streams \`MarketTick\` Protobufs.
- [ ] Uses NATS Request-Reply to await an ACK before sending the next tick.
- [ ] Absolute restriction on \`time.Now()\` or \`datetime.now()\` during backtests.
- [ ] Automated tests pass." \
  "- [ ] Create dedicated Go worker.
- [ ] Read CSV, serialize \`MarketTick\`, map CSV timestamp directly to \`event_timestamp_ms\`.
- [ ] Use \`nats.Request\` to send tick, forcing replayer to wait for synchronized response from OMS indicating the DB write is committed.
- [ ] Testing Strategy: Simulate 10,000 ticks and assert final Postgres timestamp parity against CSV."

echo "✅ All Epics and Issues have been successfully pushed to GitHub!"
echo "➡️  Next Step: Navigate to your repository on GitHub -> 'Projects' -> 'Link a Project'. Create a new V2 Board, and add your newly created Milestones to automatically populate and filter the columns."
