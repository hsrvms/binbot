package main

import (
	"context"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"

	"github.com/hsrvms/binbot/go-oms/internal/config"
	"github.com/hsrvms/binbot/go-oms/internal/database"
	"github.com/hsrvms/binbot/go-oms/internal/execution"
	"github.com/hsrvms/binbot/go-oms/internal/ingestion"
	"github.com/hsrvms/binbot/go-oms/internal/logger"
)

func main() {
	rotator, err := logger.NewLineRotator("binbot.log", 1000)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	log.SetOutput(io.MultiWriter(os.Stdout, rotator))

	cfg := config.Load()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := database.RunDatabaseMigrations(cfg.DBURL); err != nil {
		log.Fatalf("CRITICAL: Database migration failed: %v", err)
	}

	dbPool, err := pgxpool.New(ctx, cfg.DBURL)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer dbPool.Close()

	nc, err := nats.Connect(cfg.NatsURL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS at %s: %v", cfg.NatsURL, err)
	}
	defer nc.Close()

	js, err := nc.JetStream()
	if err != nil {
		log.Fatalf("Failed to bind to JetStream: %v", err)
	}

	streamName := "MARKET"
	if _, err = js.StreamInfo(streamName); err != nil {
		_, err = js.AddStream(&nats.StreamConfig{
			Name:     streamName,
			Subjects: []string{"market.data.>"},
			Storage:  nats.MemoryStorage,
			MaxAge:   0,
		})
		if err != nil {
			log.Fatalf("Failed to provision JetStream %s: %v", streamName, err)
		}
	}

	strategyStreamName := "STRATEGY"
	if _, err = js.StreamInfo(strategyStreamName); err != nil {
		_, err = js.AddStream(&nats.StreamConfig{
			Name:     strategyStreamName,
			Subjects: []string{"strategy.intent"},
			Storage:  nats.FileStorage,
			MaxAge:   0,
		})
		if err != nil {
			log.Fatalf("Failed to provision JetStream %s: %v", strategyStreamName, err)
		}
	}

	streamer := ingestion.NewStreamer(cfg.WsURL, js)
	ledger := execution.NewLedger(dbPool)
	stateServer := execution.NewStateServer(nc, ledger)

	binanceClient := execution.NewBinanceClient("", cfg.APIKey, cfg.SecretKey)

	// TODO: Build webhook
	orchestrator := execution.NewOrchestrator(js, ledger, nil, binanceClient)

	// INFO: deactivate the streamer for backtesting
	go streamer.Start(ctx)
	go stateServer.Start(ctx)
	go orchestrator.Start(ctx)

	log.Println("Binbot OMS and Ingestion Engine Running...")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Received termination signal. Shutting down gracefully...")
}
