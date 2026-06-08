package backtest

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"

	"github.com/hsrvms/binbot/go-oms/internal/pb/trading"
)

type Replayer struct {
	js nats.JetStreamContext
}

func NewReplayer(js nats.JetStreamContext) *Replayer {
	return &Replayer{js: js}
}

func (r *Replayer) Play(ctx context.Context, reader *TradeReader, symbol string, speedMultiplier float64) error {
	var lastTradeTime int64
	var lastWallTime time.Time

	subject := fmt.Sprintf("market.data.%s", symbol)
	log.Printf("Starting backtest replay for %s at speed %.2fx...", symbol, speedMultiplier)

	count := 0

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		trade, err := reader.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				log.Printf("Backtest replay complete. Processed %d trades.", count)
				return nil
			}
			return fmt.Errorf("failed to read historical trade at row %d: %w", count, err)
		}

		if speedMultiplier > 0 && lastTradeTime > 0 {
			tradeDeltaMs := trade.Timestamp - lastTradeTime
			if tradeDeltaMs > 0 {
				targetWaitDuration := time.Duration(float64(tradeDeltaMs*int64(time.Millisecond)) / speedMultiplier)
				elapsedWallTime := time.Since(lastWallTime)

				sleepTime := targetWaitDuration - elapsedWallTime
				if sleepTime > 0 {
					time.Sleep(sleepTime)
				}
			}
		}

		tick := &trading.MarketTick{
			Symbol:           symbol,
			Price:            trade.Price,
			Volume:           trade.Quantity,
			EventTimestampMs: trade.Timestamp,
		}

		data, err := proto.Marshal(tick)
		if err != nil {
			return fmt.Errorf("failed to marshal market tick: %w", err)
		}

		_, err = r.js.Publish(subject, data)
		if err != nil {
			return fmt.Errorf("failed to publish market tick to NATS: %w", err)
		}

		lastTradeTime = trade.Timestamp
		lastWallTime = time.Now()
		count++

		if count%50000 == 0 {
			log.Printf("Replayed %d historical trades...", count)
		}
	}
}
