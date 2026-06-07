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

func TestExecuteIntent_FullIOCFill(t *testing.T) {
	exchange := &MockExchange{
		IOCFillQty:  1.0,
		IOCAvgPrice: 65000.0,
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
	if ledger.RecordedQty != 1.0 {
		t.Errorf("Expected ledger to record 1.0 qty, got %f", ledger.RecordedQty)
	}
}

func TestExecuteIntent_PartialIOC_WithMarketFallback(t *testing.T) {
	exchange := &MockExchange{
		IOCFillQty:    0.4,
		IOCAvgPrice:   65000.0,
		MarketFillQty: 0.6,
		MarketPrice:   65100.0,
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
	if ledger.RecordedQty != 1.0 {
		t.Errorf("Expected ledger to record 1.0 total qty, got %f", ledger.RecordedQty)
	}
	if ledger.RecordedPrice != 65060.0 {
		t.Errorf("Expected ledger to record VWAP of 65060.0, got %f", ledger.RecordedPrice)
	}
}

func TestExecuteIntent_ExchangeFailure(t *testing.T) {
	exchange := &MockExchange{
		IOCError: errors.New("binance API timeout"),
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
	exchange := &MockExchange{}
	ledger := &MockLedger{
		Balances: map[string]float64{"BTCUSDT": 1.0},
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
