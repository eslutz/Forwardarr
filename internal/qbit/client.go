package qbit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
)

type Client struct {
	baseURL string
	user    string
	pass    string
	client  *http.Client
}

type Preferences struct {
	ListenPort int `json:"listen_port"`
}

func NewClient(baseURL, user, pass string) (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}

	client := &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		user:    user,
		pass:    pass,
		client: &http.Client{
			Jar: jar,
		},
	}

	if err := client.Login(); err != nil {
		return nil, fmt.Errorf("initial login failed: %w", err)
	}

	return client, nil
}

func (c *Client) Login() error {
	data := url.Values{}
	data.Set("username", c.user)
	data.Set("password", c.pass)

	resp, err := c.client.PostForm(c.baseURL+"/api/v2/auth/login", data)
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK || string(body) != "Ok." {
		return fmt.Errorf("login failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	slog.Debug("successfully authenticated with qBittorrent")
	return nil
}

func (c *Client) GetPort() (int, error) {
	resp, err := c.client.Get(c.baseURL + "/api/v2/app/preferences")
	if err != nil {
		return 0, fmt.Errorf("failed to get preferences: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden {
		slog.Warn("received 403, re-authenticating...")
		if err := c.Login(); err != nil {
			return 0, fmt.Errorf("re-authentication failed: %w", err)
		}
		return c.GetPort()
	}

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var prefs Preferences
	if err := json.NewDecoder(resp.Body).Decode(&prefs); err != nil {
		return 0, fmt.Errorf("failed to decode preferences: %w", err)
	}

	return prefs.ListenPort, nil
}

func (c *Client) SetPort(port int) error {
	data := map[string]interface{}{
		"listen_port": port,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL+"/api/v2/app/setPreferences", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to set preferences: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden {
		slog.Warn("received 403, re-authenticating...")
		if err := c.Login(); err != nil {
			return fmt.Errorf("re-authentication failed: %w", err)
		}
		return c.SetPort(port)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	slog.Info("successfully updated qBittorrent listening port", "port", port)
	return nil
}

func (c *Client) Ping() error {
	resp, err := c.client.Get(c.baseURL + "/api/v2/app/version")
	if err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden {
		return c.Login()
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
