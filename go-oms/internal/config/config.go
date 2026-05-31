package config

import "os"

type AppConfig struct {
	NatsURL string
	WsURL   string
}

func Load() *AppConfig {
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://nats:4222"
	}

	wsURL := os.Getenv("BINANCE_WS_URL")
	if wsURL == "" {
		wsURL = "wss://stream.testnet.binance.vision/ws/btcusdt@trade"
	}

	return &AppConfig{
		NatsURL: natsURL,
		WsURL:   wsURL,
	}
}
