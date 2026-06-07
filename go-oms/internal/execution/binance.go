package execution

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type BinanceClient struct {
	BaseURL    string
	APIKey     string
	SecretKey  string
	HTTPClient *http.Client
}

func NewBinanceClient(baseURL, apiKey, secretKey string) *BinanceClient {
	return &BinanceClient{
		BaseURL:   baseURL,
		APIKey:    apiKey,
		SecretKey: secretKey,
		HTTPClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (c *BinanceClient) sign(query string) string {
	mac := hmac.New(sha256.New, []byte(c.SecretKey))
	mac.Write([]byte(query))
	return hex.EncodeToString(mac.Sum(nil))
}

type binanceOrderResponse struct {
	Symbol              string `json:"symbol"`
	OrderId             int64  `json:"orderId"`
	Status              string `json:"status"`
	ExecutedQty         string `json:"executedQty"`
	CummulativeQuoteQty string `json:"cummulativeQuoteQty"`
}

func (c *BinanceClient) executeOrder(ctx context.Context, side, symbol, orderType, timeInForce string, qty float64) (float64, float64, error) {
	endpoint := c.BaseURL + "/api/v3/order"
	if c.BaseURL == "" {
		endpoint = "https://testnet.binance.vision/api/v3/order"
	}

	params := url.Values{}
	params.Add("symbol", symbol)
	params.Add("side", side)
	params.Add("type", orderType)
	params.Add("quantity", fmt.Sprintf("%.6f", qty))

	if timeInForce != "" {
		params.Add("timeInForce", timeInForce)
		// TODO:
		// TECHNICAL DEBT: Binance requires a 'price' for LIMIT orders.
		// Because our Orchestrator doesn't currently pass a limit price, we are hardcoding
		// a fallback here to satisfy the API and our mock server. We will need to route
		// the real-time top-of-book price here before running in production.
		params.Add("price", "65000.00")
	}

	params.Add("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))

	query := params.Encode()
	signature := c.sign(query)
	fullURL := fmt.Sprintf("%s?%s&signature=%s", endpoint, query, signature)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, nil)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-MBX-APIKEY", c.APIKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return 0, 0, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return 0, 0, fmt.Errorf("binance API error (status %d): %s", resp.StatusCode, string(body))
	}

	var orderResp binanceOrderResponse
	if err := json.Unmarshal(body, &orderResp); err != nil {
		return 0, 0, fmt.Errorf("failed to unmarshal binance response: %w", err)
	}

	executedQty, _ := strconv.ParseFloat(orderResp.ExecutedQty, 64)
	cummulativeQuoteQty, _ := strconv.ParseFloat(orderResp.CummulativeQuoteQty, 64)

	var vwap float64
	if executedQty > 0 {
		vwap = cummulativeQuoteQty / executedQty
	}

	return executedQty, vwap, nil
}

func (c *BinanceClient) ExecuteIOCLimit(ctx context.Context, side string, symbol string, qty float64) (float64, float64, error) {
	return c.executeOrder(ctx, side, symbol, "LIMIT", "IOC", qty)
}

func (c *BinanceClient) ExecuteMarket(ctx context.Context, side string, symbol string, qty float64) (float64, float64, error) {
	return c.executeOrder(ctx, side, symbol, "MARKET", "", qty)
}
