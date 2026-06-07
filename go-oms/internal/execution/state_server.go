package execution

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"

	"github.com/hsrvms/binbot/go-oms/internal/pb/trading"
)

// BalanceReader defines the contract for querying aggregated portfolio state.
type BalanceReader interface {
	GetBalances(ctx context.Context) (map[string]float64, error)
}

type StateServer struct {
	nc *nats.Conn
	db BalanceReader
}

func NewStateServer(nc *nats.Conn, db BalanceReader) *StateServer {
	return &StateServer{
		nc: nc,
		db: db,
	}
}

// GenerateStatePayload handles the internal logic of fetching data and serializing the Protobuf.
// Isolated for clean unit testing without a live NATS connection.
func (s *StateServer) GenerateStatePayload(ctx context.Context) ([]byte, error) {
	balances, err := s.db.GetBalances(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read balances from ledger: %w", err)
	}

	state := &trading.PortfolioState{
		Balances:         balances,
		StateTimestampMs: time.Now().UnixMilli(),
	}

	payload, err := proto.Marshal(state)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal PortfolioState protobuf: %w", err)
	}

	return payload, nil
}

// Start binds the server to the NATS connection and listens for synchronous requests.
func (s *StateServer) Start(ctx context.Context) error {
	if s.nc == nil {
		return fmt.Errorf("NATS connection is required for StateServer")
	}

	sub, err := s.nc.Subscribe("oms.state.get", func(msg *nats.Msg) {
		log.Println("Received portfolio state hydration request from Python Engine.")

		payload, err := s.GenerateStatePayload(ctx)
		if err != nil {
			log.Printf("CRITICAL: Failed to generate state payload: %v", err)
			msg.Respond([]byte{})
			return
		}

		if err := msg.Respond(payload); err != nil {
			log.Printf("Failed to respond to state request: %v", err)
		} else {
			log.Println("Successfully transmitted PortfolioState to Python Engine.")
		}
	})

	if err != nil {
		return fmt.Errorf("failed to subscribe to oms.state.get: %w", err)
	}
	defer sub.Unsubscribe()

	<-ctx.Done()
	log.Println("Shutting down NATS State Hydration Server...")
	return nil
}
