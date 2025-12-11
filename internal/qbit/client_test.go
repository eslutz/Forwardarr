package qbit

import (
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"
)

func TestNewClient_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/auth/login" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("Ok."))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "admin", "admin")
	if err != nil {
		t.Fatalf("NewClient() error = %v, want nil", err)
	}
	if client == nil {
		t.Fatal("NewClient() returned nil client")
	}
}

func TestNewClient_LoginFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("Fails."))
	}))
	defer server.Close()

	_, err := NewClient(server.URL, "admin", "wrongpass")
	if err == nil {
		t.Fatal("NewClient() error = nil, want error")
	}
}

func TestLogin_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Ok."))
	}))
	defer server.Close()

	client, _ := NewClient(server.URL, "admin", "admin")
	err := client.Login()
	if err != nil {
		t.Errorf("Login() error = %v, want nil", err)
	}
}

func TestLogin_Failure(t *testing.T) {
	loginAttempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		loginAttempts++
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("Fails."))
	}))
	defer server.Close()

	// Create client without initial login by using a cookiejar
	jar, _ := cookiejar.New(nil)
	client := &Client{
		baseURL: server.URL,
		user:    "admin",
		pass:    "admin",
		client: &http.Client{
			Jar: jar,
		},
	}
	
	err := client.Login()
	if err == nil {
		t.Error("Login() error = nil, want error")
	}
	if loginAttempts != 1 {
		t.Errorf("Login() attempts = %d, want 1", loginAttempts)
	}
}

func TestGetPort_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/auth/login" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("Ok."))
			return
		}
		if r.URL.Path == "/api/v2/app/preferences" {
			prefs := Preferences{ListenPort: 12345}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(prefs)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, _ := NewClient(server.URL, "admin", "admin")
	port, err := client.GetPort()
	if err != nil {
		t.Fatalf("GetPort() error = %v, want nil", err)
	}
	if port != 12345 {
		t.Errorf("GetPort() = %d, want 12345", port)
	}
}

func TestGetPort_Reauthentication(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/auth/login" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("Ok."))
			return
		}
		if r.URL.Path == "/api/v2/app/preferences" {
			callCount++
			// First call returns 403, second call succeeds
			if callCount == 1 {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			prefs := Preferences{ListenPort: 54321}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(prefs)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, _ := NewClient(server.URL, "admin", "admin")
	port, err := client.GetPort()
	if err != nil {
		t.Fatalf("GetPort() error = %v, want nil", err)
	}
	if port != 54321 {
		t.Errorf("GetPort() = %d, want 54321", port)
	}
	if callCount != 2 {
		t.Errorf("GetPort() call count = %d, want 2", callCount)
	}
}

func TestSetPort_Success(t *testing.T) {
	receivedPort := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/auth/login" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("Ok."))
			return
		}
		if r.URL.Path == "/api/v2/app/setPreferences" {
			var data map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&data)
			if port, ok := data["listen_port"].(float64); ok {
				receivedPort = int(port)
			}
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, _ := NewClient(server.URL, "admin", "admin")
	err := client.SetPort(9999)
	if err != nil {
		t.Fatalf("SetPort() error = %v, want nil", err)
	}
	if receivedPort != 9999 {
		t.Errorf("SetPort() received port = %d, want 9999", receivedPort)
	}
}

func TestSetPort_Reauthentication(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/auth/login" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("Ok."))
			return
		}
		if r.URL.Path == "/api/v2/app/setPreferences" {
			callCount++
			// First call returns 403, second call succeeds
			if callCount == 1 {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, _ := NewClient(server.URL, "admin", "admin")
	err := client.SetPort(8888)
	if err != nil {
		t.Fatalf("SetPort() error = %v, want nil", err)
	}
	if callCount != 2 {
		t.Errorf("SetPort() call count = %d, want 2", callCount)
	}
}

func TestPing_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/auth/login" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("Ok."))
			return
		}
		if r.URL.Path == "/api/v2/app/version" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("v4.5.0"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, _ := NewClient(server.URL, "admin", "admin")
	err := client.Ping()
	if err != nil {
		t.Errorf("Ping() error = %v, want nil", err)
	}
}

func TestPing_Failure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/auth/login" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("Ok."))
			return
		}
		if r.URL.Path == "/api/v2/app/version" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, _ := NewClient(server.URL, "admin", "admin")
	err := client.Ping()
	if err == nil {
		t.Error("Ping() error = nil, want error")
	}
}

func TestPing_Reauthentication(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/auth/login" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("Ok."))
			return
		}
		if r.URL.Path == "/api/v2/app/version" {
			// Return 403 to trigger re-auth
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, _ := NewClient(server.URL, "admin", "admin")
	err := client.Ping()
	// Ping only calls Login() on 403 but doesn't retry the request
	// So it should return nil (successful re-authentication)
	if err != nil {
		t.Errorf("Ping() error = %v, want nil", err)
	}
}
