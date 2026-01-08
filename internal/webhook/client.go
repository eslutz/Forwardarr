package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// Template represents the webhook payload format
type Template string

const (
	TemplateJSON    Template = "json"
	TemplateDiscord Template = "discord"
	TemplateSlack   Template = "slack"
	TemplateGotify  Template = "gotify"
)

// Client handles sending webhook notifications
type Client struct {
	url      string
	timeout  time.Duration
	template Template
	events   map[string]bool
	client   *http.Client
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
func NewClient(url string, timeout time.Duration, template Template, events []string) *Client {
	eventMap := make(map[string]bool)
	for _, event := range events {
		eventMap[strings.TrimSpace(event)] = true
	}

	return &Client{
		url:      url,
		timeout:  timeout,
		template: template,
		events:   eventMap,
		client:   &http.Client{},
	}
}

// SendPortChange sends a port change notification
func (c *Client) SendPortChange(oldPort, newPort int) error {
	event := "port_changed"

	// Check if this event is enabled
	if len(c.events) > 0 && !c.events[event] {
		slog.Debug("webhook event filtered out", "event", event)
		return nil
	}

	payload := Payload{
		Event:     event,
		Timestamp: time.Now().UTC(),
		OldPort:   oldPort,
		NewPort:   newPort,
		Message:   fmt.Sprintf("Port changed from %d to %d", oldPort, newPort),
	}

	return c.send(payload)
}

// send sends the webhook payload to the configured URL
func (c *Client) send(payload Payload) error {
	var jsonData []byte
	var err error

	// Format payload based on template
	switch c.template {
	case TemplateDiscord:
		jsonData, err = c.formatDiscord(payload)
	case TemplateSlack:
		jsonData, err = c.formatSlack(payload)
	case TemplateGotify:
		jsonData, err = c.formatGotify(payload)
	default:
		jsonData, err = json.Marshal(payload)
	}

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

	slog.Debug("sending webhook", "url", c.url, "event", payload.Event, "template", c.template)

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

// formatDiscord formats payload for Discord webhook
func (c *Client) formatDiscord(payload Payload) ([]byte, error) {
	discord := map[string]interface{}{
		"content": payload.Message,
		"embeds": []map[string]interface{}{
			{
				"title":       "Port Change Notification",
				"description": payload.Message,
				"color":       3447003, // Blue color
				"fields": []map[string]interface{}{
					{
						"name":   "Event",
						"value":  payload.Event,
						"inline": true,
					},
					{
						"name":   "Old Port",
						"value":  fmt.Sprintf("%d", payload.OldPort),
						"inline": true,
					},
					{
						"name":   "New Port",
						"value":  fmt.Sprintf("%d", payload.NewPort),
						"inline": true,
					},
				},
				"timestamp": payload.Timestamp.Format(time.RFC3339),
			},
		},
	}
	return json.Marshal(discord)
}

// formatSlack formats payload for Slack webhook
func (c *Client) formatSlack(payload Payload) ([]byte, error) {
	slack := map[string]interface{}{
		"text": payload.Message,
		"blocks": []map[string]interface{}{
			{
				"type": "section",
				"text": map[string]string{
					"type": "mrkdwn",
					"text": fmt.Sprintf("*%s*\n%s", "Port Change Notification", payload.Message),
				},
			},
			{
				"type": "section",
				"fields": []map[string]string{
					{
						"type": "mrkdwn",
						"text": fmt.Sprintf("*Event:*\n%s", payload.Event),
					},
					{
						"type": "mrkdwn",
						"text": fmt.Sprintf("*Old Port:*\n%d", payload.OldPort),
					},
					{
						"type": "mrkdwn",
						"text": fmt.Sprintf("*New Port:*\n%d", payload.NewPort),
					},
					{
						"type": "mrkdwn",
						"text": fmt.Sprintf("*Time:*\n%s", payload.Timestamp.Format(time.RFC3339)),
					},
				},
			},
		},
	}
	return json.Marshal(slack)
}

// formatGotify formats payload for Gotify webhook
func (c *Client) formatGotify(payload Payload) ([]byte, error) {
	gotify := map[string]interface{}{
		"title":    "Port Change Notification",
		"message":  payload.Message,
		"priority": 5,
		"extras": map[string]interface{}{
			"event":     payload.Event,
			"old_port":  payload.OldPort,
			"new_port":  payload.NewPort,
			"timestamp": payload.Timestamp.Format(time.RFC3339),
		},
	}
	return json.Marshal(gotify)
}
