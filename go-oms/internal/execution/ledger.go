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
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

type Ledger struct {
	db DBPool
}

func NewLedger(db DBPool) *Ledger {
	return &Ledger{db: db}
}

// GetBalances aggregates the total executed quantity for each symbol.
func (l *Ledger) GetBalances(ctx context.Context) (map[string]float64, error) {
	query := `SELECT symbol, SUM(executed_qty) FROM order_ledger GROUP BY symbol`
	rows, err := l.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrLedgerQuery, err)
	}
	defer rows.Close()

	balances := make(map[string]float64)
	for rows.Next() {
		var symbol string
		var totalQty float64
		if err := rows.Scan(&symbol, &totalQty); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrLedgerScan, err)
		}
		balances[symbol] = totalQty
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: row iteration error: %v", ErrLedgerQuery, err)
	}

	return balances, nil
}

// Record fill atomically commits a trade execution to the PostgreSQL ledger.
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
