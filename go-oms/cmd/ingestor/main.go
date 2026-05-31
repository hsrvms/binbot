package main

import (
	"context"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/hsrvms/binbot/go-oms/internal/config"
	"github.com/hsrvms/binbot/go-oms/internal/ingestion"
	"github.com/hsrvms/binbot/go-oms/internal/logger"
	"github.com/nats-io/nats.go"
)

func main() {
	rotator, err := logger.NewLineRotator("binbot.log", 1000)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	log.SetOutput(io.MultiWriter(os.Stdout, rotator))

	cfg := config.Load()

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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go streamer.Start(ctx)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Received termination signal. Shutting down gracefully...")
	cancel()
}
