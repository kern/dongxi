package dongxi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const (
	baseURL = "https://cloud.culturedcode.com/version/1"
)

// Client is an HTTP client for the Things Cloud API.
type Client struct {
	email    string
	password string
	http     *http.Client
	BaseURL  string
}

// NewClient creates a new API client.
func NewClient(email, password string) *Client {
	return &Client{
		email:    email,
		password: password,
		http:     &http.Client{},
		BaseURL:  baseURL,
	}
}

// Email returns the email associated with this client.
func (c *Client) Email() string {
	return c.email
}

func (c *Client) newRequest(method, path string, body any) (*http.Request, error) {
	u := c.BaseURL + path

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, u, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", "ThingsMac/31516502")
	req.Header.Set("App-Id", "com.culturedcode.ThingsMac")
	req.Header.Set("App-Instance-Id", "-com.culturedcode.ThingsMac")
	req.Header.Set("Schema", "301")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Charset", "UTF-8")
	req.Header.Set("Accept-Language", "en-gb")
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.Header.Set("Push-Priority", "5")
	req.Header.Set("Authorization", "Password "+c.password)

	return req, nil
}

func (c *Client) do(req *http.Request, out any) error {
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(data))
	}

	if out != nil {
		if err := json.Unmarshal(data, out); err != nil {
			return fmt.Errorf("unmarshal response: %w", err)
		}
	}

	return nil
}

// GetAccount fetches account information for the given email.
func (c *Client) GetAccount(email string) (*Account, error) {
	req, err := c.newRequest("GET", "/account/"+url.PathEscape(email), nil)
	if err != nil {
		return nil, fmt.Errorf("get account: %w", err)
	}

	var acct Account
	if err := c.do(req, &acct); err != nil {
		return nil, fmt.Errorf("get account: %w", err)
	}

	return &acct, nil
}

// GetHistory fetches history summary for the given history key.
func (c *Client) GetHistory(historyKey string) (*HistoryInfo, error) {
	req, err := c.newRequest("GET", "/history/"+url.PathEscape(historyKey), nil)
	if err != nil {
		return nil, fmt.Errorf("get history: %w", err)
	}

	var info HistoryInfo
	if err := c.do(req, &info); err != nil {
		return nil, fmt.Errorf("get history: %w", err)
	}

	return &info, nil
}

// GetHistoryItems fetches all history items, handling pagination automatically.
func (c *Client) GetHistoryItems(historyKey string) ([]map[string]any, error) {
	return c.GetHistoryItemsFrom(historyKey, 0)
}

// GetHistoryItemsFrom fetches history items starting from the given index.
func (c *Client) GetHistoryItemsFrom(historyKey string, startIndex int) ([]map[string]any, error) {
	var all []map[string]any
	start := startIndex
	for {
		path := fmt.Sprintf("/history/%s/items?start-index=%d", url.PathEscape(historyKey), start)
		req, err := c.newRequest("GET", path, nil)
		if err != nil {
			return nil, fmt.Errorf("get history items: %w", err)
		}

		var page HistoryItems
		if err := c.do(req, &page); err != nil {
			return nil, fmt.Errorf("get history items: %w", err)
		}
		if len(page.Items) == 0 {
			break
		}
		all = append(all, page.Items...)
		start += len(page.Items)
	}
	return all, nil
}

// Commit posts a batch of changes to the history.
// ancestorIndex is the current server index before this commit.
func (c *Client) Commit(historyKey string, ancestorIndex int, items map[string]CommitItem) (*CommitResponse, error) {
	path := fmt.Sprintf("/history/%s/commit?ancestor-index=%d&_cnt=%d",
		url.PathEscape(historyKey), ancestorIndex, len(items))

	req, err := c.newRequest("POST", path, items)
	if err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	var resp CommitResponse
	if err := c.do(req, &resp); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return &resp, nil
}

// ResetHistory resets the history for the given email account.
func (c *Client) ResetHistory(email string) (*ResetResponse, error) {
	path := fmt.Sprintf("/account/%s/own-history-key/reset", url.PathEscape(email))
	req, err := c.newRequest("POST", path, nil)
	if err != nil {
		return nil, fmt.Errorf("reset history: %w", err)
	}

	var resp ResetResponse
	if err := c.do(req, &resp); err != nil {
		return nil, fmt.Errorf("reset history: %w", err)
	}

	return &resp, nil
}
