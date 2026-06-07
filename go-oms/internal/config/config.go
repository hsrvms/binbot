package config

import "os"

type AppConfig struct {
	NatsURL string
	WsURL   string
	DBURL   string
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

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@postgres:5432/bot_ledger?sslmode=disable"
	}

	return &AppConfig{
		NatsURL: natsURL,
		WsURL:   wsURL,
		DBURL:   dbURL,
	}
}
