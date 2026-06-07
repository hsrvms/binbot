package execution_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hsrvms/binbot/go-oms/internal/execution"
)

func TestBinanceClient_ExecuteIOCLimit_Success(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-MBX-APIKEY") != "test_api_key" {
			t.Errorf("Expected API Key header, got: %s", r.Header.Get("X-MBX-APIKEY"))
		}

		if r.Method != http.MethodPost {
			t.Errorf("Expected POST method, got: %s", r.Method)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"symbol": "BTCUSDT",
			"orderId": 12345,
			"status": "FILLED",
			"type": "LIMIT",
			"side": "BUY",
			"executedQty": "1.5",
			"cummulativeQuoteQty": "97500.00"
		}`))
	}))
	defer mockServer.Close()

	client := execution.NewBinanceClient(mockServer.URL, "test_api_key", "test_secret_key")

	qty, price, err := client.ExecuteIOCLimit(context.Background(), "BUY", "BTCUSDT", 1.5)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if qty != 1.5 {
		t.Errorf("Expected quantity 1.5, got %f", qty)
	}

	expectedPrice := 65000.0
	if price != expectedPrice {
		t.Errorf("Expected average price %f, got %f", expectedPrice, price)
	}
}

func TestBinanceClient_HttpError(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"code": -1003, "msg": "Too many requests"}`))
	}))
	defer mockServer.Close()

	client := execution.NewBinanceClient(mockServer.URL, "test_api_key", "test_secret_key")

	_, _, err := client.ExecuteIOCLimit(context.Background(), "BUY", "BTCUSDT", 1.0)

	if err == nil {
		t.Fatal("Expected an error for HTTP 429, got nil")
	}
}
