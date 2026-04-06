package dongxi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientEmail(t *testing.T) {
	client := NewClient("user@example.com", "pass")
	if got := client.Email(); got != "user@example.com" {
		t.Errorf("Email() = %q, want %q", got, "user@example.com")
	}
}

func TestGetAccount(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/account/test@example.com") {
			t.Errorf("path = %s, want .../account/test@example.com", r.URL.Path)
		}
		// Verify headers.
		if r.Header.Get("Authorization") != "Password secret" {
			t.Errorf("Authorization = %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Schema") != "301" {
			t.Errorf("Schema = %q", r.Header.Get("Schema"))
		}
		if r.Header.Get("App-Id") != "com.culturedcode.ThingsMac" {
			t.Errorf("App-Id = %q", r.Header.Get("App-Id"))
		}

		json.NewEncoder(w).Encode(Account{
			Email:      "test@example.com",
			HistoryKey: "hk-123",
			Status:     "active",
		})
	}))
	defer srv.Close()

	client := NewClient("test@example.com", "secret")
	client.BaseURL = srv.URL + "/version/1"

	acct, err := client.GetAccount("test@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if acct.Email != "test@example.com" {
		t.Errorf("Email = %q", acct.Email)
	}
	if acct.HistoryKey != "hk-123" {
		t.Errorf("HistoryKey = %q", acct.HistoryKey)
	}
}

func TestGetAccountError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		w.Write([]byte("Unauthorized"))
	}))
	defer srv.Close()

	client := NewClient("test@example.com", "wrong")
	client.BaseURL = srv.URL + "/version/1"

	_, err := client.GetAccount("test@example.com")
	if err == nil {
		t.Fatal("expected error for 401")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("error = %q, want to contain '401'", err.Error())
	}
}

func TestGetHistory(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/history/hk-123") {
			t.Errorf("path = %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(HistoryInfo{
			LatestServerIndex:   42,
			LatestSchemaVersion: 301,
			IsEmpty:             false,
		})
	}))
	defer srv.Close()

	client := NewClient("test@example.com", "secret")
	client.BaseURL = srv.URL + "/version/1"

	info, err := client.GetHistory("hk-123")
	if err != nil {
		t.Fatal(err)
	}
	if info.LatestServerIndex != 42 {
		t.Errorf("LatestServerIndex = %d, want 42", info.LatestServerIndex)
	}
}

func TestGetHistoryItemsPagination(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		startIndex := r.URL.Query().Get("start-index")

		var resp HistoryItems
		resp.CurrentItemIndex = 3

		if startIndex == "0" {
			resp.Items = []map[string]any{
				{"task-1": map[string]any{"t": float64(0), "e": "Task6", "p": map[string]any{"tt": "First"}}},
				{"task-2": map[string]any{"t": float64(0), "e": "Task6", "p": map[string]any{"tt": "Second"}}},
			}
		} else if startIndex == "2" {
			resp.Items = []map[string]any{
				{"task-3": map[string]any{"t": float64(0), "e": "Task6", "p": map[string]any{"tt": "Third"}}},
			}
		} else {
			resp.Items = nil
		}

		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := NewClient("test@example.com", "secret")
	client.BaseURL = srv.URL + "/version/1"

	items, err := client.GetHistoryItems("hk-123")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 {
		t.Fatalf("got %d items, want 3", len(items))
	}
	if callCount != 3 {
		t.Errorf("made %d HTTP calls, want 3 (2 pages + 1 empty)", callCount)
	}
}

func TestGetHistoryItemsEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(HistoryItems{Items: nil})
	}))
	defer srv.Close()

	client := NewClient("test@example.com", "secret")
	client.BaseURL = srv.URL + "/version/1"

	items, err := client.GetHistoryItems("hk-123")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Errorf("got %d items, want 0", len(items))
	}
}

func TestCommit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/history/hk-123/commit") {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.URL.Query().Get("ancestor-index") != "10" {
			t.Errorf("ancestor-index = %s, want 10", r.URL.Query().Get("ancestor-index"))
		}
		if r.URL.Query().Get("_cnt") != "1" {
			t.Errorf("_cnt = %s, want 1", r.URL.Query().Get("_cnt"))
		}

		// Verify body.
		var body map[string]CommitItem
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if _, ok := body["task-1"]; !ok {
			t.Error("body missing task-1")
		}

		json.NewEncoder(w).Encode(CommitResponse{ServerHeadIndex: 11})
	}))
	defer srv.Close()

	client := NewClient("test@example.com", "secret")
	client.BaseURL = srv.URL + "/version/1"

	items := map[string]CommitItem{
		"task-1": {
			T: ItemTypeModify,
			E: EntityTask,
			P: map[string]any{"tt": "Hello"},
		},
	}

	resp, err := client.Commit("hk-123", 10, items)
	if err != nil {
		t.Fatal(err)
	}
	if resp.ServerHeadIndex != 11 {
		t.Errorf("ServerHeadIndex = %d, want 11", resp.ServerHeadIndex)
	}
}

func TestResetHistory(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/account/test@example.com/own-history-key/reset") {
			t.Errorf("path = %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(ResetResponse{NewHistoryKey: "new-hk-456"})
	}))
	defer srv.Close()

	client := NewClient("test@example.com", "secret")
	client.BaseURL = srv.URL + "/version/1"

	resp, err := client.ResetHistory("test@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if resp.NewHistoryKey != "new-hk-456" {
		t.Errorf("NewHistoryKey = %q, want %q", resp.NewHistoryKey, "new-hk-456")
	}
}

func TestCommitError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(409)
		w.Write([]byte("Conflict"))
	}))
	defer srv.Close()

	client := NewClient("test@example.com", "secret")
	client.BaseURL = srv.URL + "/version/1"

	_, err := client.Commit("hk-123", 10, map[string]CommitItem{})
	if err == nil {
		t.Fatal("expected error for 409")
	}
	if !strings.Contains(err.Error(), "409") {
		t.Errorf("error = %q, want to contain '409'", err.Error())
	}
}

// --- newRequest error paths ---

func TestNewRequestMarshalError(t *testing.T) {
	client := NewClient("test@example.com", "secret")
	// A channel cannot be marshaled to JSON.
	_, err := client.newRequest("GET", "/test", make(chan int))
	if err == nil {
		t.Fatal("expected marshal error")
	}
	if !strings.Contains(err.Error(), "marshal request body") {
		t.Errorf("error = %q, want 'marshal request body'", err.Error())
	}
}

func TestNewRequestInvalidURL(t *testing.T) {
	client := NewClient("test@example.com", "secret")
	client.BaseURL = "http://example.com/\x7f"
	_, err := client.newRequest("GET", "/test", nil)
	if err == nil {
		t.Fatal("expected URL error")
	}
	if !strings.Contains(err.Error(), "create request") {
		t.Errorf("error = %q, want 'create request'", err.Error())
	}
}

// --- do error paths ---

// errReader is an io.ReadCloser that always returns an error on Read.
type errReader struct{}

func (e errReader) Read([]byte) (int, error) { return 0, errors.New("read boom") }
func (e errReader) Close() error             { return nil }

// errTransport is an http.RoundTripper that returns a response with a broken body.
type errTransport struct{ resp *http.Response }

func (t errTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return t.resp, nil
}

// failTransport is an http.RoundTripper that always returns an error.
type failTransport struct{}

func (failTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("connection refused")
}

func TestDoConnectionError(t *testing.T) {
	client := NewClient("test@example.com", "secret")
	client.http = &http.Client{Transport: failTransport{}}

	req, _ := http.NewRequest("GET", "http://localhost/test", nil)
	err := client.do(req, nil)
	if err == nil {
		t.Fatal("expected connection error")
	}
	if !strings.Contains(err.Error(), "execute request") {
		t.Errorf("error = %q, want 'execute request'", err.Error())
	}
}

func TestDoReadBodyError(t *testing.T) {
	client := NewClient("test@example.com", "secret")
	client.http = &http.Client{
		Transport: errTransport{resp: &http.Response{
			StatusCode: 200,
			Body:       errReader{},
		}},
	}

	req, _ := http.NewRequest("GET", "http://localhost/test", nil)
	err := client.do(req, nil)
	if err == nil {
		t.Fatal("expected read body error")
	}
	if !strings.Contains(err.Error(), "read response body") {
		t.Errorf("error = %q, want 'read response body'", err.Error())
	}
}

func TestDoUnmarshalError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	client := NewClient("test@example.com", "secret")
	client.BaseURL = srv.URL

	req, _ := client.newRequest("GET", "/test", nil)
	var result Account
	err := client.do(req, &result)
	if err == nil {
		t.Fatal("expected unmarshal error")
	}
	if !strings.Contains(err.Error(), "unmarshal response") {
		t.Errorf("error = %q, want 'unmarshal response'", err.Error())
	}
}

func TestDoNilOut(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("anything"))
	}))
	defer srv.Close()

	client := NewClient("test@example.com", "secret")
	client.BaseURL = srv.URL

	req, _ := client.newRequest("GET", "/test", nil)
	// out is nil, so no unmarshal should happen.
	err := client.do(req, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- GetAccount newRequest error ---

func TestGetAccountNewRequestError(t *testing.T) {
	client := NewClient("test@example.com", "secret")
	client.BaseURL = "http://example.com/\x7f"

	_, err := client.GetAccount("test@example.com")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "get account") {
		t.Errorf("error = %q, want 'get account'", err.Error())
	}
}

// --- GetHistory error paths ---

func TestGetHistoryNewRequestError(t *testing.T) {
	client := NewClient("test@example.com", "secret")
	client.BaseURL = "http://example.com/\x7f"

	_, err := client.GetHistory("hk-123")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "get history") {
		t.Errorf("error = %q, want 'get history'", err.Error())
	}
}

func TestGetHistoryDoError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte("Internal Server Error"))
	}))
	defer srv.Close()

	client := NewClient("test@example.com", "secret")
	client.BaseURL = srv.URL + "/version/1"

	_, err := client.GetHistory("hk-123")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "get history") {
		t.Errorf("error = %q, want 'get history'", err.Error())
	}
}

func TestGetHistoryItemsFromStartIndex(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startIndex := r.URL.Query().Get("start-index")
		var resp HistoryItems
		if startIndex == "5" {
			resp.Items = []map[string]any{
				{"task-6": map[string]any{"t": float64(0), "e": "Task6", "p": map[string]any{"tt": "Sixth"}}},
			}
		} else {
			resp.Items = nil
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := NewClient("test@example.com", "secret")
	client.BaseURL = srv.URL + "/version/1"

	items, err := client.GetHistoryItemsFrom("hk-123", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("got %d items, want 1", len(items))
	}
}

// --- GetHistoryItems error paths ---

func TestGetHistoryItemsNewRequestError(t *testing.T) {
	client := NewClient("test@example.com", "secret")
	client.BaseURL = "http://example.com/\x7f"

	_, err := client.GetHistoryItems("hk-123")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "get history items") {
		t.Errorf("error = %q, want 'get history items'", err.Error())
	}
}

func TestGetHistoryItemsDoError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte("Server Error"))
	}))
	defer srv.Close()

	client := NewClient("test@example.com", "secret")
	client.BaseURL = srv.URL + "/version/1"

	_, err := client.GetHistoryItems("hk-123")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "get history items") {
		t.Errorf("error = %q, want 'get history items'", err.Error())
	}
}

// --- Commit newRequest error ---

func TestCommitNewRequestError(t *testing.T) {
	client := NewClient("test@example.com", "secret")
	client.BaseURL = "http://example.com/\x7f"

	_, err := client.Commit("hk-123", 10, map[string]CommitItem{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "commit") {
		t.Errorf("error = %q, want 'commit'", err.Error())
	}
}

// --- ResetHistory error paths ---

func TestResetHistoryNewRequestError(t *testing.T) {
	client := NewClient("test@example.com", "secret")
	client.BaseURL = "http://example.com/\x7f"

	_, err := client.ResetHistory("test@example.com")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "reset history") {
		t.Errorf("error = %q, want 'reset history'", err.Error())
	}
}

func TestResetHistoryDoError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte("Server Error"))
	}))
	defer srv.Close()

	client := NewClient("test@example.com", "secret")
	client.BaseURL = srv.URL + "/version/1"

	_, err := client.ResetHistory("test@example.com")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "reset history") {
		t.Errorf("error = %q, want 'reset history'", err.Error())
	}
}

// --- newRequest with body (happy path, verifying body is set) ---

func TestNewRequestWithBody(t *testing.T) {
	client := NewClient("test@example.com", "secret")
	body := map[string]string{"key": "value"}
	req, err := client.newRequest("POST", "/test", body)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := io.ReadAll(req.Body)
	if !strings.Contains(string(data), "key") {
		t.Errorf("body = %q, want to contain 'key'", string(data))
	}
}
