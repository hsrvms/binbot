package execution

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
)

type DBPool interface {
	Begin(context.Context) (pgx.Tx, error)
}

type Ledger struct {
	db DBPool
}

func NewLedger(db DBPool) *Ledger {
	return &Ledger{db: db}
}

func (l *Ledger) RecordFill(ctx context.Context, symbol string, qty, price float64, reasoning string) error {
	tx, err := l.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrTransactionBegin, err)
	}

	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && rbErr.Error() != "tx is closed" {
			log.Printf("CRITICAL: Ledger rollback failed: %v", rbErr)
		}
	}()

	query := `
		INSERT INTO order_ledger (symbol, executed_price, executed_qty, strategy_reasoning, timestamp_ms)
		VALUES ($1, $2, $3, $4, $5)
	`

	timestamp := time.Now().UnixMilli()
	_, err = tx.Exec(ctx, query, symbol, price, qty, reasoning, timestamp)
	if err != nil {
		return fmt.Errorf("%w: insert failed: %v", ErrLedgerCommit, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("%w: tx commit failed: %v", ErrLedgerCommit, err)
	}

	return nil
}
