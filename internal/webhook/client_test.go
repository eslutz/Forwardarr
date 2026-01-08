package webhook

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	url := "http://example.com/webhook"
	timeout := 5 * time.Second
	template := TemplateJSON
	events := []string{"port_changed"}

	client := NewClient(url, timeout, template, events)

	if client.url != url {
		t.Errorf("client.url = %v, want %v", client.url, url)
	}
	if client.timeout != timeout {
		t.Errorf("client.timeout = %v, want %v", client.timeout, timeout)
	}
	if client.template != template {
		t.Errorf("client.template = %v, want %v", client.template, template)
	}
	if len(client.events) != len(events) {
		t.Errorf("client.events length = %d, want %d", len(client.events), len(events))
	}
	if client.client == nil {
		t.Error("client.client is nil, want non-nil")
	}
}

func TestSendPortChange_Success(t *testing.T) {
	var receivedPayload Payload
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("request method = %v, want POST", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %v, want application/json", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("User-Agent") != "Forwardarr-Webhook/1.0" {
			t.Errorf("User-Agent = %v, want Forwardarr-Webhook/1.0", r.Header.Get("User-Agent"))
		}

		if err := json.NewDecoder(r.Body).Decode(&receivedPayload); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, 5*time.Second, TemplateJSON, []string{"port_changed"})
	err := client.SendPortChange(8080, 9090)

	if err != nil {
		t.Errorf("SendPortChange() error = %v, want nil", err)
	}

	if receivedPayload.Event != "port_changed" {
		t.Errorf("payload.Event = %v, want port_changed", receivedPayload.Event)
	}
	if receivedPayload.OldPort != 8080 {
		t.Errorf("payload.OldPort = %d, want 8080", receivedPayload.OldPort)
	}
	if receivedPayload.NewPort != 9090 {
		t.Errorf("payload.NewPort = %d, want 9090", receivedPayload.NewPort)
	}
	if receivedPayload.Message != "Port changed from 8080 to 9090" {
		t.Errorf("payload.Message = %v, want 'Port changed from 8080 to 9090'", receivedPayload.Message)
	}
	if receivedPayload.Timestamp.IsZero() {
		t.Error("payload.Timestamp is zero, want non-zero")
	}
}

func TestSendPortChange_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(server.URL, 5*time.Second, TemplateJSON, []string{"port_changed"})
	err := client.SendPortChange(8080, 9090)

	if err == nil {
		t.Error("SendPortChange() error = nil, want error")
	}
}

func TestSendPortChange_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, 10*time.Millisecond, TemplateJSON, []string{"port_changed"})
	err := client.SendPortChange(8080, 9090)

	if err == nil {
		t.Error("SendPortChange() error = nil, want timeout error")
	}
}

func TestSendPortChange_InvalidURL(t *testing.T) {
	client := NewClient("http://[::1]:namedport", 5*time.Second, TemplateJSON, []string{"port_changed"})
	err := client.SendPortChange(8080, 9090)

	if err == nil {
		t.Error("SendPortChange() error = nil, want error")
	}
}

func TestSendPortChange_NonOKStatusCodes(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{"bad request", http.StatusBadRequest},
		{"unauthorized", http.StatusUnauthorized},
		{"forbidden", http.StatusForbidden},
		{"not found", http.StatusNotFound},
		{"internal server error", http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client := NewClient(server.URL, 5*time.Second, TemplateJSON, []string{"port_changed"})
			err := client.SendPortChange(8080, 9090)

			if err == nil {
				t.Errorf("SendPortChange() error = nil, want error for status %d", tt.statusCode)
			}
		})
	}
}

func TestSendPortChange_SuccessStatusCodes(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{"ok", http.StatusOK},
		{"created", http.StatusCreated},
		{"accepted", http.StatusAccepted},
		{"no content", http.StatusNoContent},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client := NewClient(server.URL, 5*time.Second, TemplateJSON, []string{"port_changed"})
			err := client.SendPortChange(8080, 9090)

			if err != nil {
				t.Errorf("SendPortChange() error = %v, want nil for status %d", err, tt.statusCode)
			}
		})
	}
}

func TestTemplateFormats(t *testing.T) {
tests := []struct {
name     string
template Template
validate func(*testing.T, map[string]interface{})
}{
{
name:     "discord template",
template: TemplateDiscord,
validate: func(t *testing.T, payload map[string]interface{}) {
if payload["content"] == nil {
t.Error("Discord payload missing 'content' field")
}
if payload["embeds"] == nil {
t.Error("Discord payload missing 'embeds' field")
}
},
},
{
name:     "slack template",
template: TemplateSlack,
validate: func(t *testing.T, payload map[string]interface{}) {
if payload["text"] == nil {
t.Error("Slack payload missing 'text' field")
}
if payload["blocks"] == nil {
t.Error("Slack payload missing 'blocks' field")
}
},
},
{
name:     "gotify template",
template: TemplateGotify,
validate: func(t *testing.T, payload map[string]interface{}) {
if payload["title"] == nil {
t.Error("Gotify payload missing 'title' field")
}
if payload["message"] == nil {
t.Error("Gotify payload missing 'message' field")
}
if payload["extras"] == nil {
t.Error("Gotify payload missing 'extras' field")
}
},
},
{
name:     "json template",
template: TemplateJSON,
validate: func(t *testing.T, payload map[string]interface{}) {
if payload["event"] == nil {
t.Error("JSON payload missing 'event' field")
}
if payload["old_port"] == nil {
t.Error("JSON payload missing 'old_port' field")
}
if payload["new_port"] == nil {
t.Error("JSON payload missing 'new_port' field")
}
},
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
var receivedPayload map[string]interface{}
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
if err := json.NewDecoder(r.Body).Decode(&receivedPayload); err != nil {
t.Errorf("failed to decode request body: %v", err)
}
w.WriteHeader(http.StatusOK)
}))
defer server.Close()

client := NewClient(server.URL, 5*time.Second, tt.template, []string{"port_changed"})
err := client.SendPortChange(8080, 9090)

if err != nil {
t.Errorf("SendPortChange() error = %v, want nil", err)
}

tt.validate(t, receivedPayload)
})
}
}

func TestEventFiltering(t *testing.T) {
tests := []struct {
name            string
events          []string
shouldSend      bool
}{
{
name:       "port_changed event enabled",
events:     []string{"port_changed"},
shouldSend: true,
},
{
name:       "port_changed with other events",
events:     []string{"port_changed", "sync_error"},
shouldSend: true,
},
{
name:       "port_changed event disabled",
events:     []string{"sync_error"},
shouldSend: false,
},
{
name:       "empty events list sends all",
events:     []string{},
shouldSend: true,
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
callCount := 0
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
callCount++
w.WriteHeader(http.StatusOK)
}))
defer server.Close()

client := NewClient(server.URL, 5*time.Second, TemplateJSON, tt.events)
err := client.SendPortChange(8080, 9090)

if err != nil {
t.Errorf("SendPortChange() error = %v, want nil", err)
}

if tt.shouldSend && callCount == 0 {
t.Error("expected webhook to be sent but it wasn't")
}
if !tt.shouldSend && callCount > 0 {
t.Error("expected webhook not to be sent but it was")
}
})
}
}
