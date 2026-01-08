package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// Client handles sending webhook notifications
type Client struct {
	url     string
	timeout time.Duration
	client  *http.Client
}

// Payload represents the webhook notification payload
type Payload struct {
	Event     string    `json:"event"`
	Timestamp time.Time `json:"timestamp"`
	OldPort   int       `json:"old_port"`
	NewPort   int       `json:"new_port"`
	Message   string    `json:"message"`
}

// NewClient creates a new webhook client
func NewClient(url string, timeout time.Duration) *Client {
	return &Client{
		url:     url,
		timeout: timeout,
		client:  &http.Client{},
	}
}

// SendPortChange sends a port change notification
func (c *Client) SendPortChange(oldPort, newPort int) error {
	payload := Payload{
		Event:     "port_changed",
		Timestamp: time.Now().UTC(),
		OldPort:   oldPort,
		NewPort:   newPort,
		Message:   fmt.Sprintf("Port changed from %d to %d", oldPort, newPort),
	}

	return c.send(payload)
}

// send sends the webhook payload to the configured URL
func (c *Client) send(payload Payload) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Forwardarr-Webhook/1.0")

	slog.Debug("sending webhook", "url", c.url, "event", payload.Event)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Warn("failed to close webhook response body", "error", err)
		}
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned non-2xx status: %d", resp.StatusCode)
	}

	slog.Info("webhook sent successfully", "url", c.url, "status", resp.StatusCode)
	return nil
}
