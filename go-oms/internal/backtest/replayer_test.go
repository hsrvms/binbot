package backtest_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"

	"github.com/hsrvms/binbot/go-oms/internal/backtest"
	"github.com/hsrvms/binbot/go-oms/internal/pb/trading"
)

type MockJetStream struct {
	nats.JetStreamContext
	PublishedData [][]byte
	LastSubject   string
}

func (m *MockJetStream) Publish(subj string, data []byte, opts ...nats.PubOpt) (*nats.PubAck, error) {
	m.LastSubject = subj
	m.PublishedData = append(m.PublishedData, data)
	return &nats.PubAck{}, nil
}

func TestReplayer_Play_MaxSpeed(t *testing.T) {
	csvData := `42398412,61350.00,0.01250000,766.875,1780879900000,true,true
42398413,61351.00,0.05000000,3067.55,1780879901000,false,true`

	reader := backtest.NewTradeReader(strings.NewReader(csvData))
	mockJS := &MockJetStream{}
	replayer := backtest.NewReplayer(mockJS)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := replayer.Play(ctx, reader, "btcusdt", 0.0)
	if err != nil {
		t.Fatalf("Expected replayer to finish cleanly, got: %v", err)
	}

	if len(mockJS.PublishedData) != 2 {
		t.Errorf("Expected 2 messages published, got %d", len(mockJS.PublishedData))
	}

	if mockJS.LastSubject != "market.data.btcusdt" {
		t.Errorf("Expected subject market.data.btcusdt, got %s", mockJS.LastSubject)
	}

	var tick trading.MarketTick
	err = proto.Unmarshal(mockJS.PublishedData[0], &tick)
	if err != nil {
		t.Fatalf("Failed to unmarshal published payload: %v", err)
	}

	if tick.Price != 61350.00 {
		t.Errorf("Expected price 61350.00, got %f", tick.Price)
	}
	if tick.Volume != 0.0125 {
		t.Errorf("Expected volume 0.0125, got %f", tick.Volume)
	}
	if tick.EventTimestampMs != 1780879900000 {
		t.Errorf("Expected EventTimestampMs 1780879900000, got %d", tick.EventTimestampMs)
	}
}

func TestReplayer_Play_PacedSpeed(t *testing.T) {
	csvData := `42398412,61350.00,0.01250000,766.875,1780879900000,true,true
42398413,61351.00,0.05000000,3067.55,1780879901000,false,true`

	reader := backtest.NewTradeReader(strings.NewReader(csvData))
	mockJS := &MockJetStream{}
	replayer := backtest.NewReplayer(mockJS)

	startTime := time.Now()

	err := replayer.Play(context.Background(), reader, "btcusdt", 10.0)
	if err != nil {
		t.Fatalf("Expected clean finish, got: %v", err)
	}

	elapsed := time.Since(startTime)

	if elapsed < 80*time.Millisecond {
		t.Errorf("Replayer did not sleep long enough for pacing. Expected ~100ms, took %v", elapsed)
	}
}
