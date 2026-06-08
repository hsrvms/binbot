package backtest

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strconv"
)

var (
	ErrInvalidColumnCount = errors.New("invalid column count")
	ErrInvalidPrice       = errors.New("invalid price: must be positive")
	ErrInvalidQuantity    = errors.New("invalid quantity: must be positive")
)

type HistoricalTrade struct {
	TradeID      int64
	Price        float64
	Quantity     float64
	QuoteQty     float64
	Timestamp    int64
	IsBuyerMaker bool
	IsBestMatch  bool
}

type TradeReader struct {
	csvReader *csv.Reader
	isFirst   bool
}

func NewTradeReader(r io.Reader) *TradeReader {
	return &TradeReader{
		csvReader: csv.NewReader(r),
		isFirst:   true,
	}
}

func (tr *TradeReader) Read() (*HistoricalTrade, error) {
	for {
		record, err := tr.csvReader.Read()
		if err != nil {
			return nil, err
		}

		if tr.isFirst {
			tr.isFirst = false
			if record[0] == "id" || record[0] == "trade_id" {
				continue
			}
		}
		return ParseTradeRow(record)
	}
}

func ParseTradeRow(record []string) (*HistoricalTrade, error) {
	if len(record) < 7 {
		return nil, fmt.Errorf("%w: expected at least 7 columns, got %d", ErrInvalidColumnCount, len(record))
	}

	tradeID, err := strconv.ParseInt(record[0], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse trade_id: %w", err)
	}

	price, err := strconv.ParseFloat(record[1], 64)
	if err != nil || price <= 0 {
		if err == nil {
			err = ErrInvalidPrice
		}
		return nil, fmt.Errorf("failed to parse price: %w", err)
	}

	quantity, err := strconv.ParseFloat(record[2], 64)
	if err != nil || quantity <= 0 {
		if err == nil {
			err = ErrInvalidQuantity
		}
		return nil, fmt.Errorf("failed to parse quantity: %w", err)
	}

	quoteQty, err := strconv.ParseFloat(record[3], 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse quote_qty: %w", err)
	}

	timestamp, err := strconv.ParseInt(record[4], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse timestamp: %w", err)
	}

	isBuyerMaker, err := strconv.ParseBool(record[5])
	if err != nil {
		return nil, fmt.Errorf("failed to parse is_buyer_maker: %w", err)
	}

	isBestMatch, err := strconv.ParseBool(record[6])
	if err != nil {
		return nil, fmt.Errorf("failed to parse is_best_match: %w", err)
	}

	return &HistoricalTrade{
		TradeID:      tradeID,
		Price:        price,
		Quantity:     quantity,
		QuoteQty:     quoteQty,
		Timestamp:    timestamp,
		IsBuyerMaker: isBuyerMaker,
		IsBestMatch:  isBestMatch,
	}, nil
}
