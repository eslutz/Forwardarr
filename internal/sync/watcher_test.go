package sync

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/eslutz/forwardarr/internal/qbit"
	"github.com/eslutz/forwardarr/internal/webhook"
)

func TestReadPortFromFile_Success(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()
	portFile := filepath.Join(tmpDir, "forwarded_port")

	// Write a valid port
	err := os.WriteFile(portFile, []byte("12345"), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	w := &Watcher{portFile: portFile}
	port, err := w.readPortFromFile()
	if err != nil {
		t.Fatalf("readPortFromFile() error = %v, want nil", err)
	}
	if port != 12345 {
		t.Errorf("readPortFromFile() = %d, want 12345", port)
	}
}

func newTestQbitServer(t *testing.T, initialPort int, getStatus, setStatus int) (*httptest.Server, *int, *int, *int) {
	t.Helper()

	// Use heap-allocated variables so pointers remain valid
	port := new(int)
	*port = initialPort
	setPortCalls := new(int)
	getPortCalls := new(int)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/auth/login":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("Ok."))
		case "/api/v2/app/preferences":
			*getPortCalls++
			status := http.StatusOK
			if getStatus != 0 {
				status = getStatus
			}

			if status != http.StatusOK {
				w.WriteHeader(status)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(qbit.Preferences{ListenPort: *port})
		case "/api/v2/app/setPreferences":
			*setPortCalls++
			status := http.StatusOK
			if setStatus != 0 {
				status = setStatus
			}

			if status != http.StatusOK {
				w.WriteHeader(status)
				return
			}

			err := r.ParseForm()
			if err != nil {
				t.Errorf("ParseForm error: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			// qBittorrent API expects the preferences as JSON in the 'json' form field
			jsonStr := r.Form.Get("json")
			if jsonStr != "" {
				var prefs map[string]int
				if err := json.Unmarshal([]byte(jsonStr), &prefs); err != nil {
					t.Errorf("json.Unmarshal error: %v", err)
				} else if newPort, ok := prefs["listen_port"]; ok {
					*port = newPort
				}
			}
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	return server, port, getPortCalls, setPortCalls
}

func TestWatcherSyncPortUpdatesPort(t *testing.T) {
	tmpDir := t.TempDir()
	portFile := filepath.Join(tmpDir, "forwarded_port")
	if err := os.WriteFile(portFile, []byte("9090"), 0644); err != nil {
		t.Fatalf("failed to write port file: %v", err)
	}

	server, port, _, setPortCalls := newTestQbitServer(t, 8080, 0, 0)
	defer server.Close()

	client, err := qbit.NewClient(server.URL, "user", "pass")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	watcher := &Watcher{portFile: portFile, qbitClient: client, webhookClient: nil}
	if err := watcher.syncPort(); err != nil {
		t.Fatalf("syncPort() error = %v", err)
	}

	if *port != 9090 {
		t.Fatalf("qBittorrent port = %d, want 9090", *port)
	}
	if *setPortCalls != 1 {
		t.Fatalf("SetPreferences call count = %d, want 1", *setPortCalls)
	}
	if watcher.lastPort != 9090 {
		t.Fatalf("watcher.lastPort = %d, want 9090", watcher.lastPort)
	}
}

func TestWatcherSyncPortAlreadyInSync(t *testing.T) {
	tmpDir := t.TempDir()
	portFile := filepath.Join(tmpDir, "forwarded_port")
	if err := os.WriteFile(portFile, []byte("1234"), 0644); err != nil {
		t.Fatalf("failed to write port file: %v", err)
	}

	server, port, _, setPortCalls := newTestQbitServer(t, 1234, 0, 0)
	defer server.Close()

	client, err := qbit.NewClient(server.URL, "user", "pass")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	watcher := &Watcher{portFile: portFile, qbitClient: client, webhookClient: nil}
	if err := watcher.syncPort(); err != nil {
		t.Fatalf("syncPort() error = %v", err)
	}

	if *port != 1234 {
		t.Fatalf("qBittorrent port changed = %d, want 1234", *port)
	}
	if *setPortCalls != 0 {
		t.Fatalf("SetPreferences call count = %d, want 0", *setPortCalls)
	}
}

func TestWatcherSyncPortGetPortError(t *testing.T) {
	tmpDir := t.TempDir()
	portFile := filepath.Join(tmpDir, "forwarded_port")
	if err := os.WriteFile(portFile, []byte("5555"), 0644); err != nil {
		t.Fatalf("failed to write port file: %v", err)
	}

	server, _, _, setPortCalls := newTestQbitServer(t, 1111, http.StatusInternalServerError, 0)
	defer server.Close()

	client, err := qbit.NewClient(server.URL, "user", "pass")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	watcher := &Watcher{portFile: portFile, qbitClient: client, webhookClient: nil}
	if err := watcher.syncPort(); err == nil {
		t.Fatal("syncPort() error = nil, want error")
	}
	if *setPortCalls != 0 {
		t.Fatalf("SetPreferences call count = %d, want 0", *setPortCalls)
	}
}

func TestWatcherSyncPortSetPortError(t *testing.T) {
	tmpDir := t.TempDir()
	portFile := filepath.Join(tmpDir, "forwarded_port")
	if err := os.WriteFile(portFile, []byte("6000"), 0644); err != nil {
		t.Fatalf("failed to write port file: %v", err)
	}

	server, port, _, setPortCalls := newTestQbitServer(t, 4000, 0, http.StatusInternalServerError)
	defer server.Close()

	client, err := qbit.NewClient(server.URL, "user", "pass")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	watcher := &Watcher{portFile: portFile, qbitClient: client, webhookClient: nil}
	if err := watcher.syncPort(); err == nil {
		t.Fatal("syncPort() error = nil, want error")
	}

	if *port != 4000 {
		t.Fatalf("qBittorrent port changed = %d, want 4000", *port)
	}
	if *setPortCalls != 1 {
		t.Fatalf("SetPreferences call count = %d, want 1", *setPortCalls)
	}
}

func TestReadPortFromFile_WithWhitespace(t *testing.T) {
	tmpDir := t.TempDir()
	portFile := filepath.Join(tmpDir, "forwarded_port")

	// Write a port with whitespace
	err := os.WriteFile(portFile, []byte("  54321  \n"), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	w := &Watcher{portFile: portFile}
	port, err := w.readPortFromFile()
	if err != nil {
		t.Fatalf("readPortFromFile() error = %v, want nil", err)
	}
	if port != 54321 {
		t.Errorf("readPortFromFile() = %d, want 54321", port)
	}
}

func TestReadPortFromFile_FileNotFound(t *testing.T) {
	w := &Watcher{portFile: "/nonexistent/path/port"}
	_, err := w.readPortFromFile()
	if err == nil {
		t.Error("readPortFromFile() error = nil, want error")
	}
}

func TestReadPortFromFile_InvalidPort(t *testing.T) {
	tmpDir := t.TempDir()
	portFile := filepath.Join(tmpDir, "forwarded_port")

	// Write invalid port content
	err := os.WriteFile(portFile, []byte("not-a-number"), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	w := &Watcher{portFile: portFile}
	_, err = w.readPortFromFile()
	if err == nil {
		t.Error("readPortFromFile() error = nil, want error")
	}
}

func TestReadPortFromFile_PortOutOfRange(t *testing.T) {
	tests := []struct {
		name string
		port string
	}{
		{"port too low", "0"},
		{"port too high", "65536"},
		{"negative port", "-1"},
		{"very high port", "99999"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			portFile := filepath.Join(tmpDir, "forwarded_port")

			err := os.WriteFile(portFile, []byte(tt.port), 0644)
			if err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			w := &Watcher{portFile: portFile}
			_, err = w.readPortFromFile()
			if err == nil {
				t.Errorf("readPortFromFile() with port %s: error = nil, want error", tt.port)
			}
		})
	}
}

func TestReadPortFromFile_ValidEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		port     string
		expected int
	}{
		{"minimum valid port", "1", 1},
		{"maximum valid port", "65535", 65535},
		{"common port", "8080", 8080},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			portFile := filepath.Join(tmpDir, "forwarded_port")

			err := os.WriteFile(portFile, []byte(tt.port), 0644)
			if err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			w := &Watcher{portFile: portFile}
			port, err := w.readPortFromFile()
			if err != nil {
				t.Errorf("readPortFromFile() error = %v, want nil", err)
			}
			if port != tt.expected {
				t.Errorf("readPortFromFile() = %d, want %d", port, tt.expected)
			}
		})
	}
}

func TestWatcherSyncPortWithWebhook(t *testing.T) {
	tmpDir := t.TempDir()
	portFile := filepath.Join(tmpDir, "forwarded_port")
	if err := os.WriteFile(portFile, []byte("7070"), 0644); err != nil {
		t.Fatalf("failed to write port file: %v", err)
	}

	// Setup qBittorrent test server
	qbitServer, port, _, setPortCalls := newTestQbitServer(t, 5050, 0, 0)
	defer qbitServer.Close()

	qbitClient, err := qbit.NewClient(qbitServer.URL, "user", "pass")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// Setup webhook test server
	var webhookCalled bool
	var receivedOldPort, receivedNewPort int
	webhookServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		webhookCalled = true
		var payload struct {
			Event   string `json:"event"`
			OldPort int    `json:"old_port"`
			NewPort int    `json:"new_port"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("failed to decode webhook payload: %v", err)
		}
		receivedOldPort = payload.OldPort
		receivedNewPort = payload.NewPort
		w.WriteHeader(http.StatusOK)
	}))
	defer webhookServer.Close()

	// Create webhook client
	webhookClient := webhook.NewClient(webhookServer.URL, 5*time.Second, webhook.TemplateJSON, []string{"port_changed"})

	watcher := &Watcher{portFile: portFile, qbitClient: qbitClient, webhookClient: webhookClient}
	if err := watcher.syncPort(); err != nil {
		t.Fatalf("syncPort() error = %v", err)
	}

	if *port != 7070 {
		t.Errorf("qBittorrent port = %d, want 7070", *port)
	}
	if *setPortCalls != 1 {
		t.Errorf("SetPreferences call count = %d, want 1", *setPortCalls)
	}
	if !webhookCalled {
		t.Error("webhook was not called")
	}
	if receivedOldPort != 5050 {
		t.Errorf("webhook old_port = %d, want 5050", receivedOldPort)
	}
	if receivedNewPort != 7070 {
		t.Errorf("webhook new_port = %d, want 7070", receivedNewPort)
	}
}
