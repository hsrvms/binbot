package ingestion

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hsrvms/binbot/go-oms/internal/pb/trading"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
)

type Publisher interface {
	Publish(subj string, data []byte, opts ...nats.PubOpt) (*nats.PubAck, error)
}

type FlexInt int64

func (fi *FlexInt) UnmarshalJSON(b []byte) error {
	s := string(b)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return err
	}
	*fi = FlexInt(i)
	return nil
}

type BinanceTrade struct {
	EventType string  `json:"e"`
	EventTime FlexInt `json:"E"`
	Symbol    string  `json:"s"`
	Price     string  `json:"p"`
	Quantity  string  `json:"q"`
}

type Streamer struct {
	wsURL string
	pub   Publisher
}

func NewStreamer(wsURL string, pub Publisher) *Streamer {
	return &Streamer{
		wsURL: wsURL,
		pub:   pub,
	}
}

func (s *Streamer) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			log.Println("Context cancelled, shutting down ingestion streamer.")
			return
		default:
			log.Printf("Connecting to Binance WebSocket: %s", s.wsURL)
			if err := s.connectAndRead(ctx); err != nil {
				log.Printf("Ingestion halted: %v. Reconnecting in 3 seconds...", err)
				time.Sleep(3 * time.Second)
			}
		}
	}
}

func (s *Streamer) connectAndRead(ctx context.Context) error {
	dialCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	conn, _, err := websocket.DefaultDialer.DialContext(dialCtx, s.wsURL, nil)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrWebSocketDial, err)
	}
	defer conn.Close()

	readCtx, readCancel := context.WithCancel(ctx)
	defer readCancel()

	go func() {
		<-readCtx.Done()
		conn.Close()
	}()

	log.Println("Successfully connected to Binance Websocket.")

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("%w: %v", ErrWebSocketRead, err)
		}

		var rawTrade BinanceTrade
		if err := json.Unmarshal(message, &rawTrade); err != nil {
			log.Printf("%v: json unmarshal error: %v", ErrInvalidPayload, err)
			continue
		}

		price, _ := strconv.ParseFloat(rawTrade.Price, 64)
		volume, _ := strconv.ParseFloat(rawTrade.Quantity, 64)

		tick := &trading.MarketTick{
			Symbol:           rawTrade.Symbol,
			Price:            price,
			Volume:           volume,
			EventTimestampMs: int64(rawTrade.EventTime),
		}

		payload, err := proto.Marshal(tick)
		if err != nil {
			log.Printf("%v: protobuf marshal error: %v", ErrInvalidPayload, err)
			continue
		}

		subject := fmt.Sprintf("market.data.%s", rawTrade.Symbol)
		_, err = s.pub.Publish(subject, payload)
		if err != nil {
			log.Printf("%v: subject %s: %v", ErrPublishStream, subject, err)
		}
	}
}
