package backtest_test

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/hsrvms/binbot/go-oms/internal/backtest"
)

func TestParseTradeRow_Valid(t *testing.T) {
	row := []string{"42398412", "61350.00", "0.01250000", "766.875", "1780879900000", "true", "true"}
	trade, err := backtest.ParseTradeRow(row)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if trade.TradeID != 42398412 {
		t.Errorf("Expected TradeID 42398412, got %d", trade.TradeID)
	}
	if trade.Price != 61350.00 {
		t.Errorf("Expected Price 61350.00, got %f", trade.Price)
	}
	if trade.Quantity != 0.01250000 {
		t.Errorf("Expected Quantity 0.0125, got %f", trade.Quantity)
	}
	if trade.Timestamp != 1780879900000 {
		t.Errorf("Expected Timestamp 1780879900000, got %d", trade.Timestamp)
	}
	if !trade.IsBuyerMaker {
		t.Errorf("Expected IsBuyerMaker to be true")
	}
}

func TestParseTradeRow_InvalidData(t *testing.T) {
	tests := []struct {
		name        string
		row         []string
		expectedErr error
	}{
		{
			name:        "Missing Columns",
			row:         []string{"42398412", "61350.00"},
			expectedErr: backtest.ErrInvalidColumnCount,
		},
		{
			name:        "Negative Price",
			row:         []string{"42398412", "-61350.00", "0.0125", "766.875", "1780879900000", "true", "true"},
			expectedErr: backtest.ErrInvalidPrice,
		},
		{
			name:        "Zero Quantity",
			row:         []string{"42398412", "61350.00", "0.0000", "0.000", "1780879900000", "true", "true"},
			expectedErr: backtest.ErrInvalidQuantity,
		},
		{
			name:        "Malformed Timestamp",
			row:         []string{"42398412", "61350.00", "0.0125", "766.875", "not_a_time", "true", "true"},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := backtest.ParseTradeRow(tt.row)
			if err == nil {
				t.Fatal("Expected an error, got nil")
			}
			if tt.expectedErr != nil && !errors.Is(err, tt.expectedErr) {
				t.Errorf("Expected error to wrap %v, got %v", tt.expectedErr, err)
			}
		})
	}
}

func TestTradeReader_StreamIntegration(t *testing.T) {
	csvData := `42398412,61350.00,0.01250000,766.875,1780879900000,true,true
42398413,61351.00,0.05000000,3067.55,1780879900100,false,true`

	reader := backtest.NewTradeReader(strings.NewReader(csvData))

	trade1, err := reader.Read()
	if err != nil {
		t.Fatalf("Failed to read first row: %v", err)
	}
	if trade1.TradeID != 42398412 {
		t.Errorf("Expected first trade ID 42398412, got %d", trade1.TradeID)
	}

	trade2, err := reader.Read()
	if err != nil {
		t.Fatalf("Failed to read second row: %v", err)
	}
	if trade2.TradeID != 42398413 {
		t.Errorf("Expected second trade ID 42398413, got %d", trade2.TradeID)
	}

	_, err = reader.Read()
	if err != io.EOF {
		t.Errorf("Expected EOF, got %v", err)
	}
}
