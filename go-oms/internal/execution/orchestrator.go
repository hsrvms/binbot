package execution

import (
	"context"
	"fmt"
	"log"

	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"

	"github.com/hsrvms/binbot/go-oms/internal/pb/trading"
)

// ExchangeClient defines the contract for Binance execution, allowing mocks in CI.
type ExchangeClient interface {
	ExecuteIOCLimit(ctx context.Context, symbol string, qty float64) (float64, float64, error)
	ExecuteMarket(ctx context.Context, symbol string, qty float64) (float64, float64, error)
}

// LedgerWriter defines the contract for database commits.
type LedgerWriter interface {
	RecordFill(ctx context.Context, symbol string, qty, price float64, reasoning string) error
}

// WebhookBroadcaster defines the contract for outbound alerts.
type WebhookBroadcaster interface {
	Broadcast(message string) error
}

type Orchestrator struct {
	js       nats.JetStreamContext
	ledger   LedgerWriter
	webhook  WebhookBroadcaster
	exchange ExchangeClient
}

// NewOrchestrator now accepts interfaces, making it fully testable via Dependency Injection.
func NewOrchestrator(js nats.JetStreamContext, ledger LedgerWriter, webhook WebhookBroadcaster, exchange ExchangeClient) *Orchestrator {
	return &Orchestrator{
		js:       js,
		ledger:   ledger,
		webhook:  webhook,
		exchange: exchange,
	}
}

// ExecuteIntent isolates the Binance logic: IOC limit order with a Market order fallback.
func (o *Orchestrator) ExecuteIntent(ctx context.Context, intent *trading.IntentSignal) error {
	iocQty, iocPrice, err := o.exchange.ExecuteIOCLimit(ctx, intent.Symbol, intent.TargetExposure)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrExchangeIOC, err)
	}

	totalQty := iocQty
	totalValue := iocQty * iocPrice

	remainingQty := intent.TargetExposure - iocQty
	if remainingQty > 0.00000001 {
		mktQty, mktPrice, err := o.exchange.ExecuteMarket(ctx, intent.Symbol, remainingQty)
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

	if err := o.ledger.RecordFill(ctx, intent.Symbol, totalQty, vwap, intent.StrategyReasoning); err != nil {

		return err
	}

	return nil
}

// Start binds the NATS stream to the execution logic.
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
			msg.Nak()
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
