package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"golang.org/x/time/rate"

	"v12-handy-cap-bot/model"
)

const (
	initialBackoff = 1 * time.Second
	maxBackoff     = 60 * time.Second
	backoffFactor  = 2
	rateLimit      = 30 // req/sec/IP
)

type WSClient struct {
	mu      sync.Mutex
	url     string
	conn    *websocket.Conn
	limiter *rate.Limiter
	assets  map[int][]*model.AssetsResponse
}

func NewWSClient(url string) (*WSClient, error) {
	c := &WSClient{
		url:     url,
		limiter: rate.NewLimiter(rate.Limit(rateLimit), rateLimit),
	}
	if err := c.connect(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *WSClient) connect() error {
	if c.conn != nil {
		c.conn.Close()
	}
	conn, _, err := websocket.DefaultDialer.Dial(c.url, nil)
	if err != nil {
		return fmt.Errorf("ws dial: %w", err)
	}
	c.conn = conn
	log.Println("WebSocket connected")
	return nil
}

// GetAssets returns a copy of the last fetched assets map.
func (c *WSClient) GetAssets() map[int][]*model.AssetsResponse {
	c.mu.Lock()
	defer c.mu.Unlock()
	cp := make(map[int][]*model.AssetsResponse, len(c.assets))
	for k, v := range c.assets {
		cp[k] = v
	}
	return cp
}

func (c *WSClient) reconnect(ctx context.Context) {
	backoff := initialBackoff
	for {
		select {
		case <-ctx.Done():
			log.Println("reconnect cancelled")
			return
		default:
		}

		log.Printf("reconnecting in %s...", backoff)

		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			log.Println("reconnect cancelled")
			return
		case <-timer.C:
		}

		if err := c.connect(); err != nil {
			log.Printf("reconnect failed: %v", err)
			backoff *= backoffFactor
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}
		return
	}
}

func (c *WSClient) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		c.conn.Close()
	}
}

func (c *WSClient) rpcCall(method string) (json.RawMessage, error) {
	if err := c.limiter.Wait(context.Background()); err != nil {
		return nil, fmt.Errorf("rate limit: %w", err)
	}

	req := model.JsonRPCRequest{
		JsonRPC: "2.0",
		ID:      uuid.NewString(),
		Method:  method,
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	err := c.conn.WriteJSON(req)
	if err != nil {
		c.reconnect(context.Background())
		return nil, fmt.Errorf("ws write: %w", err)
	}

	var resp model.JsonRPCResponse
	err = c.conn.ReadJSON(&resp)
	if err != nil {
		c.reconnect(context.Background())
		return nil, fmt.Errorf("ws read: %w", err)
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("rpc error %d: %s", resp.Error.Code, resp.Error.Message)
	}

	return resp.Result, nil
}

func (c *WSClient) FetchCaps() ([]model.SLCapsStatus, error) {
	result, err := c.rpcCall("caps")
	if err != nil {
		return nil, err
	}

	var caps []model.SLCapsStatus
	if err := json.Unmarshal(result, &caps); err != nil {
		return nil, fmt.Errorf("unmarshal caps: %w", err)
	}

	return caps, nil
}

func (c *WSClient) FetchAssets() (map[int][]*model.AssetsResponse, error) {
	result, err := c.rpcCall("assets")
	if err != nil {
		return nil, err
	}

	var raw map[string][]*model.AssetsResponse
	if err := json.Unmarshal(result, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal assets: %w", err)
	}

	assets := convertChainKeys(raw)

	c.mu.Lock()
	c.assets = assets
	c.mu.Unlock()
	return assets, nil
}

func convertChainKeys(raw map[string][]*model.AssetsResponse) map[int][]*model.AssetsResponse {
	assets := make(map[int][]*model.AssetsResponse, len(raw))
	for k, v := range raw {
		chainID, err := strconv.Atoi(k)
		if err != nil {
			log.Printf("invalid chain ID key %q: %v", k, err)
			continue
		}
		assets[chainID] = v
	}
	return assets
}
