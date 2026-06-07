package execution

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"

	"github.com/hsrvms/binbot/go-oms/internal/pb/trading"
)

type ExchangeClient interface {
	ExecuteIOCLimit(ctx context.Context, side string, symbol string, qty float64) (float64, float64, error)
	ExecuteMarket(ctx context.Context, side string, symbol string, qty float64) (float64, float64, error)
}

type LedgerWriter interface {
	RecordFill(ctx context.Context, symbol string, qty, price float64, reasoning string) error
	GetBalances(ctx context.Context) (map[string]float64, error)
}

type WebhookBroadcaster interface {
	Broadcast(message string) error
}

type Orchestrator struct {
	js       nats.JetStreamContext
	ledger   LedgerWriter
	webhook  WebhookBroadcaster
	exchange ExchangeClient
}

func NewOrchestrator(js nats.JetStreamContext, ledger LedgerWriter, webhook WebhookBroadcaster, exchange ExchangeClient) *Orchestrator {
	return &Orchestrator{
		js:       js,
		ledger:   ledger,
		webhook:  webhook,
		exchange: exchange,
	}
}

func (o *Orchestrator) ExecuteIntent(ctx context.Context, intent *trading.IntentSignal) error {
	balances, err := o.ledger.GetBalances(ctx)
	if err != nil {
		return fmt.Errorf("failed to get balances for delta calculation: %w", err)
	}
	currentQty := balances[intent.Symbol]

	delta := intent.TargetExposure - currentQty
	if math.Abs(delta) < 0.00000001 {
		log.Printf("Target exposure for %s already met. Ignoring intent.", intent.Symbol)
		return nil
	}

	side := "BUY"
	if delta < 0 {
		side = "SELL"
	}
	tradeQty := math.Abs(delta)

	iocQty, iocPrice, err := o.exchange.ExecuteIOCLimit(ctx, side, intent.Symbol, tradeQty)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrExchangeIOC, err)
	}

	totalQty := iocQty
	totalValue := iocQty * iocPrice

	remainingQty := tradeQty - iocQty
	if remainingQty > 0.00000001 {
		mktQty, mktPrice, err := o.exchange.ExecuteMarket(ctx, side, intent.Symbol, remainingQty)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrExchangeMarket, err)
		}
		totalQty += mktQty
		totalValue += (mktQty * mktPrice)
	}

	if totalQty == 0 {
		return ErrZeroQuantity
	}

	vwap := totalValue / totalQty

	ledgerQty := totalQty
	if side == "SELL" {
		ledgerQty = -totalQty
	}

	if err := o.ledger.RecordFill(ctx, intent.Symbol, ledgerQty, vwap, intent.StrategyReasoning); err != nil {
		return err
	}

	return nil
}

func (o *Orchestrator) Start(ctx context.Context) error {
	sub, err := o.js.Subscribe("strategy.intent", func(msg *nats.Msg) {
		var intent trading.IntentSignal
		if err := proto.Unmarshal(msg.Data, &intent); err != nil {
			log.Printf("%v: %v", ErrIntentUnmarshal, err)
			msg.Nak()
			return
		}

		log.Printf("Received Intent: Target %f %s | Reason: %s", intent.TargetExposure, intent.Symbol, intent.StrategyReasoning)

		if err := o.ExecuteIntent(ctx, &intent); err != nil {
			log.Printf("CRITICAL Execution Failure: %v", err)
			msg.NakWithDelay(5 * time.Second)
			return
		}

		if o.webhook != nil {
			alertMsg := fmt.Sprintf("✅ **Executed:** %s | Target: %f | Reason: %s", intent.Symbol, intent.TargetExposure, intent.StrategyReasoning)
			if err := o.webhook.Broadcast(alertMsg); err != nil {
				log.Printf("Webhook broadcast failed: %v", err)
			}
		}

		msg.Ack()
	}, nats.Durable("OMS_EXECUTION_WORKER"), nats.ManualAck())

	if err != nil {
		return fmt.Errorf("%w: %v", ErrIntentSubscription, err)
	}
	defer sub.Unsubscribe()

	<-ctx.Done()
	log.Println("Shutting down OMS execution loop...")
	return nil
}
