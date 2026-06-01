package execution_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hsrvms/binbot/go-oms/internal/execution"
)

func TestWebhookClient_Broadcast_Success(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request. got %s", r.Method)
		}

		var payload map[string]string
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Failed to decode payload: %v", err)
		}

		if payload["content"] != "Test Alert" {
			t.Errorf("Expected content 'Test Alert', got '%s'", payload["content"])
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer mockServer.Close()

	client := execution.NewWebhookClient(mockServer.URL)

	err := client.Broadcast("Test Alert")

	if err != nil {
		t.Fatalf("Expected no error on successful broadcast, got: %v", err)
	}
}

func TestWebhookClient_Broadcast_Non200Response(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer mockServer.Close()

	client := execution.NewWebhookClient(mockServer.URL)

	err := client.Broadcast("Failing Alert")

	if err == nil {
		t.Fatal("Expected an error for non-200 response, but got nil")
	}
}
