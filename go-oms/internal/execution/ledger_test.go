package execution_test

import (
	"context"
	"errors"
	"testing"

	"github.com/hsrvms/binbot/go-oms/internal/execution"
	"github.com/pashagolub/pgxmock/v4"
)

func TestLedger_RecordFill_AtomicCommit_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mock.Close()

	ledger := execution.NewLedger(mock)
	ctx := context.Background()

	mock.ExpectBegin()

	mock.ExpectExec("INSERT INTO trades").
		WithArgs("BTCUSDT", 65000.0, 1.0, "FILLED", pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	mock.ExpectExec("INSERT INTO balances").
		WithArgs("BTCUSDT", 1.0).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	mock.ExpectCommit()

	err = ledger.RecordFill(ctx, "BTCUSDT", 1.0, 65000.0, "Golden Cross")

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled database expectations: %v", err)
	}
}

func TestLedger_RecordFill_RollbackOnQueryFailure(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to open mock db: %v", err)
	}
	defer mock.Close()

	ledger := execution.NewLedger(mock)
	ctx := context.Background()

	mock.ExpectBegin()

	mock.ExpectExec("INSERT INTO trades").
		WithArgs("BTCUSDT", 65000.0, 1.0, "FILLED", pgxmock.AnyArg()).
		WillReturnError(errors.New("simulated constraint violation"))

	mock.ExpectRollback()

	err = ledger.RecordFill(ctx, "BTCUSDT", 1.0, 65000.0, "Golden Cross")

	if err == nil {
		t.Fatal("Expected an error due to query failure, got nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled database expectations: %v", err)
	}
}
