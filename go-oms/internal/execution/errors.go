package execution

import "errors"

var (
	ErrTransactionBegin     = errors.New("failed to begin database transaction")
	ErrLedgerCommit         = errors.New("failed to commit ledger execution")
	ErrWebhookBroadcast     = errors.New("failed to broadcast webhook payload")
	ErrMissingWebhookConfig = errors.New("webhook URL is not configured")

	ErrExchangeIOC        = errors.New("exchange failed to execute IOC limit order")
	ErrExchangeMarket     = errors.New("exchange failed to execute market fallback order")
	ErrZeroQuantity       = errors.New("execution resolved with zero quantity filled")
	ErrIntentUnmarshal    = errors.New("failed to unmarshal intent signal")
	ErrIntentSubscription = errors.New("failed to subscribe to strategy intents")
)
