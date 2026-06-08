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

const MaxTestnetUSDTSize = 500.0

func (o *Orchestrator) ExecuteIntent(ctx context.Context, intent *trading.IntentSignal) error {
	// 1. Fetch current database/ledger balances to check actual state
	balances, err := o.ledger.GetBalances(ctx)
	if err != nil {
		return fmt.Errorf("failed to retrieve ledger balances: %w", err)
	}

	currentBTC := balances[intent.Symbol]

	currentPrice := 61500.0

	var targetBTC float64
	if intent.TargetExposure > 0.0 {
		targetBTC = MaxTestnetUSDTSize / currentPrice
	} else {
		targetBTC = 0.0
	}

	tradeDelta := targetBTC - currentBTC

	if math.Abs(tradeDelta) < 0.0001 {
		log.Printf("Current position satisfies target exposure for %s. Skipping execution.", intent.Symbol)
		return nil
	}

	var side string
	var finalQty float64

	if tradeDelta > 0 {
		side = "BUY"
		finalQty = tradeDelta
	} else {
		side = "SELL"
		finalQty = math.Abs(tradeDelta)
	}

	log.Printf("Translating exposure change to order: %s %f %s at ~%.2f USDT", side, finalQty, intent.Symbol, currentPrice)

	mktQty, mktPrice, err := o.exchange.ExecuteMarket(ctx, side, intent.Symbol, finalQty)
	if err != nil {
		return fmt.Errorf("exchange failed to execute order: %w", err)
	}

	ledgerQty := mktQty
	if side == "SELL" {
		ledgerQty = -mktQty
	}

	if err := o.ledger.RecordFill(ctx, intent.Symbol, ledgerQty, mktPrice, intent.StrategyReasoning); err != nil {
		return fmt.Errorf("failed to log execution to database: %w", err)
	}

	log.Printf("Successfully synchronized ledger. Position change committed for %f %s.", ledgerQty, intent.Symbol)
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
	}, nats.Durable("OMS_EXECUTION_WORKER"), nats.ManualAck(), nats.AckWait(10*time.Minute))

	if err != nil {
		return fmt.Errorf("%w: %v", ErrIntentSubscription, err)
	}
	defer sub.Unsubscribe()

	<-ctx.Done()
	log.Println("Shutting down OMS execution loop...")
	return nil
}
