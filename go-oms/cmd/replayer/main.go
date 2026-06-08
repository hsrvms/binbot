package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/nats-io/nats.go"

	"github.com/hsrvms/binbot/go-oms/internal/backtest"
	"github.com/hsrvms/binbot/go-oms/internal/config"
)

func main() {
	filePath := flag.String("file", "", "Path to the historical Binance trades CSV file")
	symbol := flag.String("symbol", "btcusdt", "The market symbol being replayed")
	speed := flag.Float64("speed", 0.0, "Playback speed multiplier (0 = max speed, 1 = real-time, 10 = 10x)")
	flag.Parse()

	if *filePath == "" {
		log.Fatal("Error: You must provide a path to a CSV file using the -file flag.")
	}

	file, err := os.Open(*filePath)
	if err != nil {
		log.Fatalf("Failed to open file: %v", err)
	}
	defer file.Close()

	cfg := config.Load()

	nc, err := nats.Connect(cfg.NatsURL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	js, err := nc.JetStream()
	if err != nil {
		log.Fatalf("Failed to bind to JetStream: %v", err)
	}

	ctx := context.Background()
	reader := backtest.NewTradeReader(file)
	replayer := backtest.NewReplayer(js)

	if err := replayer.Play(ctx, reader, *symbol, *speed); err != nil {
		log.Fatalf("Backtest execution failed: %v", err)
	}
}
