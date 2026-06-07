package execution_test

import (
	"context"
	"errors"
	"testing"

	"github.com/hsrvms/binbot/go-oms/internal/execution"
	"github.com/hsrvms/binbot/go-oms/internal/pb/trading"
	"google.golang.org/protobuf/proto"
)

type MockBalanceReader struct {
	Balances map[string]float64
	Err      error
}

func (m *MockBalanceReader) GetBalances(ctx context.Context) (map[string]float64, error) {
	return m.Balances, m.Err
}

func TestStateServer_GeneratePayload_Success(t *testing.T) {
	mockDB := &MockBalanceReader{
		Balances: map[string]float64{
			"BTCUSDT": 1.5,
			"ETHUSDT": 10.0,
		},
		Err: nil,
	}

	server := execution.NewStateServer(nil, mockDB)

	payload, err := server.GenerateStatePayload(context.Background())

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	var state trading.PortfolioState
	if err := proto.Unmarshal(payload, &state); err != nil {
		t.Fatalf("Failed to unmarshal generated payload: %v", err)
	}

	if state.Balances["BTCUSDT"] != 1.5 {
		t.Errorf("Expected BTCUSDT balance to be 1.5, got %f", state.Balances["BTCUSDT"])
	}
	if state.StateTimestampMs == 0 {
		t.Error("Expected StateTimestampMs to be populated, got 0")
	}
}

func TestStateServer_GeneratePayload_DBFailure(t *testing.T) {
	mockDB := &MockBalanceReader{
		Err: errors.New("connection timeout"),
	}

	server := execution.NewStateServer(nil, mockDB)

	_, err := server.GenerateStatePayload(context.Background())

	if err == nil {
		t.Fatal("Expected an error due to database failure, got nil")
	}
}
