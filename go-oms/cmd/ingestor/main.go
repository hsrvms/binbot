package main

import (
	"context"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/hsrvms/binbot/go-oms/internal/config"
	"github.com/hsrvms/binbot/go-oms/internal/execution"
	"github.com/hsrvms/binbot/go-oms/internal/ingestion"
	"github.com/hsrvms/binbot/go-oms/internal/logger"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
)

// --- TEMPORARY MOCK FOR MVP END-TO-END TESTING ---
type MockBinance struct{}

func (m *MockBinance) ExecuteIOCLimit(ctx context.Context, side string, symbol string, qty float64) (float64, float64, error) {
	log.Printf("[MOCK BINANCE] Executed %s IOC Limit: %f %s @ $65000.00", side, qty, symbol)
	return qty, 65000.0, nil
}

func (m *MockBinance) ExecuteMarket(ctx context.Context, side string, symbol string, qty float64) (float64, float64, error) {
	log.Printf("[MOCK BINANCE] Executed %s Market Fallback: %f %s @ $65100.00", side, qty, symbol)
	return qty, 65100.0, nil
}

// -------------------------------------------------

func main() {
	rotator, err := logger.NewLineRotator("binbot.log", 1000)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	log.SetOutput(io.MultiWriter(os.Stdout, rotator))

	cfg := config.Load()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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
	_, err = js.StreamInfo(streamName)
	if err != nil {
		log.Printf("Stream %s not found, provisioning...", streamName)
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

	streamer := ingestion.NewStreamer(cfg.WsURL, js)
	ledger := execution.NewLedger(dbPool)
	stateServer := execution.NewStateServer(nc, ledger)

	// TODO: Build exchange and webhook
	mockExchange := &MockBinance{}
	orchestrator := execution.NewOrchestrator(js, ledger, nil, mockExchange)

	go streamer.Start(ctx)
	go stateServer.Start(ctx)
	go orchestrator.Start(ctx)

	log.Println("Binbot OMS and Igestion Engine Running...")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Received termination signal. Shutting down gracefully...")
}
