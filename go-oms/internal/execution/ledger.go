package execution

import (
	"context"
	"fmt"
	"log"
	"math"
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

func (l *Ledger) GetBalances(ctx context.Context) (map[string]float64, error) {
	query := `SELECT asset, amount FROM balances`
	rows, err := l.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrLedgerQuery, err)
	}
	defer rows.Close()

	balances := make(map[string]float64)
	for rows.Next() {
		var asset string
		var amount float64
		if err := rows.Scan(&asset, &amount); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrLedgerScan, err)
		}
		balances[asset] = amount
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: row iteration error: %v", ErrLedgerQuery, err)
	}

	return balances, nil
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

	timestamp := time.Now().UnixMilli()

	tradeQuery := `
		INSERT INTO trades (symbol, price, quantity, status, event_timestamp_ms)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err = tx.Exec(ctx, tradeQuery, symbol, price, math.Abs(qty), "FILLED", timestamp)
	if err != nil {
		return fmt.Errorf("%w: trade insert failed: %v", ErrLedgerCommit, err)
	}

	balanceQuery := `
		INSERT INTO balances (asset, amount)
		VALUES ($1, $2)
		ON CONFLICT (asset)
		DO UPDATE SET amount = balances.amount + EXCLUDED.amount, updated_at = CURRENT_TIMESTAMP
	`
	_, err = tx.Exec(ctx, balanceQuery, symbol, qty)
	if err != nil {
		return fmt.Errorf("%w: balance upsert failed: %v", ErrLedgerCommit, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("%w: tx commit failed: %v", ErrLedgerCommit, err)
	}

	return nil
}
