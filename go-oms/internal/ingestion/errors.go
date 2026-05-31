package ingestion

import "errors"

var (
	ErrWebSocketDial  = errors.New("failed to dial websocket connection")
	ErrWebSocketRead  = errors.New("failed to read from websocket")
	ErrInvalidPayload = errors.New("failed to parse or validate trade payload")
	ErrPublishStream  = errors.New("failed to publish payload to jetstream")
)
