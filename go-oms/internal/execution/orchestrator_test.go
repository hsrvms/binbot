package execution_test

import (
	"context"
	"errors"
	"testing"

	"github.com/hsrvms/binbot/go-oms/internal/execution"
	"github.com/hsrvms/binbot/go-oms/internal/pb/trading"
)

type MockExchange struct {
	IOCFillQty    float64
	IOCAvgPrice   float64
	IOCError      error
	MarketFillQty float64
	MarketPrice   float64
	MarketError   error
}

func (m *MockExchange) ExecuteIOCLimit(ctx context.Context, side string, symbol string, qty float64) (float64, float64, error) {
	return m.IOCFillQty, m.IOCAvgPrice, m.IOCError
}

func (m *MockExchange) ExecuteMarket(ctx context.Context, side string, symbol string, qty float64) (float64, float64, error) {
	return m.MarketFillQty, m.MarketPrice, m.MarketError
}

type MockLedger struct {
	RecordedSymbol string
	RecordedQty    float64
	RecordedPrice  float64
	Balances       map[string]float64
}

func (m *MockLedger) RecordFill(ctx context.Context, symbol string, qty, price float64, reasoning string) error {
	m.RecordedSymbol = symbol
	m.RecordedQty = qty
	m.RecordedPrice = price
	return nil
}

func (m *MockLedger) GetBalances(ctx context.Context) (map[string]float64, error) {
	if m.Balances == nil {
		return map[string]float64{}, nil
	}
	return m.Balances, nil
}

func TestExecuteIntent_MarketBuy_Success(t *testing.T) {
	expectedQty := 500.0 / 61500.0
	exchange := &MockExchange{
		MarketFillQty: expectedQty,
		MarketPrice:   61500.0,
	}
	ledger := &MockLedger{
		Balances: map[string]float64{"BTCUSDT": 0.0},
	}
	orchestrator := execution.NewOrchestrator(nil, ledger, nil, exchange)

	intent := &trading.IntentSignal{
		Symbol:            "BTCUSDT",
		TargetExposure:    1.0,
		StrategyReasoning: "Golden Cross",
	}

	err := orchestrator.ExecuteIntent(context.Background(), intent)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if ledger.RecordedQty != expectedQty {
		t.Errorf("Expected ledger to record %f qty, got %f", expectedQty, ledger.RecordedQty)
	}
}

func TestExecuteIntent_MarketSell_Success(t *testing.T) {
	expectedQty := 500.0 / 61500.0
	exchange := &MockExchange{
		MarketFillQty: expectedQty,
		MarketPrice:   61500.0,
	}
	// Setup: We already own the fractional amount
	ledger := &MockLedger{
		Balances: map[string]float64{"BTCUSDT": expectedQty},
	}
	orchestrator := execution.NewOrchestrator(nil, ledger, nil, exchange)

	intent := &trading.IntentSignal{
		Symbol:            "BTCUSDT",
		TargetExposure:    0.0,
		StrategyReasoning: "Death Cross",
	}

	err := orchestrator.ExecuteIntent(context.Background(), intent)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	// Assert: Sells are recorded as negative quantities in the ledger
	if ledger.RecordedQty != -expectedQty {
		t.Errorf("Expected ledger to record %f total qty, got %f", -expectedQty, ledger.RecordedQty)
	}
}

func TestExecuteIntent_ExchangeFailure(t *testing.T) {
	exchange := &MockExchange{
		MarketError: errors.New("binance API timeout"),
	}
	ledger := &MockLedger{
		Balances: map[string]float64{"BTCUSDT": 0.0},
	}
	orchestrator := execution.NewOrchestrator(nil, ledger, nil, exchange)

	intent := &trading.IntentSignal{Symbol: "BTCUSDT", TargetExposure: 1.0}

	err := orchestrator.ExecuteIntent(context.Background(), intent)

	if err == nil {
		t.Fatal("Expected an error due to exchange failure, got nil")
	}
}

func TestExecuteIntent_ZeroDelta_IgnoresExecution(t *testing.T) {
	expectedQty := 500.0 / 61500.0
	exchange := &MockExchange{}
	// Setup: We already own the fractional amount, and the intent asks for 1.0 exposure
	ledger := &MockLedger{
		Balances: map[string]float64{"BTCUSDT": expectedQty},
	}
	orchestrator := execution.NewOrchestrator(nil, ledger, nil, exchange)

	intent := &trading.IntentSignal{
		Symbol:            "BTCUSDT",
		TargetExposure:    1.0,
		StrategyReasoning: "Golden Cross",
	}

	err := orchestrator.ExecuteIntent(context.Background(), intent)

	if err != nil {
		t.Fatalf("Expected no error for ignored intent, got: %v", err)
	}
	if ledger.RecordedQty != 0 {
		t.Errorf("Ledger should not have been called, but recorded qty: %f", ledger.RecordedQty)
	}
}
