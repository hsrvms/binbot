package ingestion

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/nats-io/nats.go"
)

type MockPublisher struct {
	PublishedSubjects []string
}

func (m *MockPublisher) Publish(subj string, data []byte, opts ...nats.PubOpt) (*nats.PubAck, error) {
	m.PublishedSubjects = append(m.PublishedSubjects, subj)
	return &nats.PubAck{}, nil
}

func TestStreamer_connectAndRead(t *testing.T) {
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()

		mockTrade := `{"E":1620000000000,"s":"BTCUSDT","p":"50000.00","q":"0.1"}`
		c.WriteMessage(websocket.TextMessage, []byte(mockTrade))

		for {
			if _, _, err := c.ReadMessage(); err != nil {
				break
			}
		}
	}))
	defer server.Close()

	wsURL := strings.Replace(server.URL, "http", "ws", 1)
	mockPub := &MockPublisher{}
	streamer := NewStreamer(wsURL, mockPub)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := streamer.connectAndRead(ctx)

	if err != nil && err != context.DeadlineExceeded {
		t.Errorf("Expected DeadlineExceeded, got: %v", err)
	}

	if len(mockPub.PublishedSubjects) != 1 {
		t.Fatalf("Expected exactly 1 published message, got %d", len(mockPub.PublishedSubjects))
	}

	expectedSubject := "market.data.BTCUSDT"
	if mockPub.PublishedSubjects[0] != expectedSubject {
		t.Errorf("Expected subject %s, got %s", expectedSubject, mockPub.PublishedSubjects[0])
	}
}
