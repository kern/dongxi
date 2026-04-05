package cmd

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/kern/dongxi/dongxi"
	"github.com/spf13/cobra"
)

// loginMockClient implements CloudClient for login tests.
type loginMockClient struct {
	account *dongxi.Account
	err     error
}

func (m *loginMockClient) GetAccount(email string) (*dongxi.Account, error) {
	return m.account, m.err
}
func (m *loginMockClient) GetHistory(historyKey string) (*dongxi.HistoryInfo, error) {
	return nil, nil
}
func (m *loginMockClient) Commit(historyKey string, ancestorIndex int, items map[string]dongxi.CommitItem) (*dongxi.CommitResponse, error) {
	return nil, nil
}
func (m *loginMockClient) ResetHistory(email string) (*dongxi.ResetResponse, error) {
	return nil, nil
}
func (m *loginMockClient) GetHistoryItems(historyKey string) ([]map[string]any, error) {
	return nil, nil
}

// overrideLoginDeps replaces newClientFunc and saveConfigFunc for a test,
// restoring originals via t.Cleanup.
func overrideLoginDeps(t *testing.T, client CloudClient, saveFn func(*dongxi.Config) error) {
	t.Helper()
	origClient := newClientFunc
	origSave := saveConfigFunc
	newClientFunc = func(email, password string) CloudClient { return client }
	saveConfigFunc = saveFn
	t.Cleanup(func() {
		newClientFunc = origClient
		saveConfigFunc = origSave
	})
}

// runLoginWith executes runLogin with the given email/password, capturing
// stdout into the returned buffer.
func runLoginWith(t *testing.T, email, password string) (*bytes.Buffer, error) {
	t.Helper()
	origEmail, origPassword := flagEmail, flagPassword
	flagEmail = email
	flagPassword = password
	t.Cleanup(func() {
		flagEmail = origEmail
		flagPassword = origPassword
	})

	buf := new(bytes.Buffer)
	cmd := &cobra.Command{}
	cmd.SetOut(buf)

	err := runLogin(cmd, nil)
	return buf, err
}

func TestRunLoginSuccess(t *testing.T) {
	mc := &loginMockClient{
		account: &dongxi.Account{HistoryKey: "hk-123", Email: "a@b.com", Status: "active"},
	}
	var savedCfg *dongxi.Config
	overrideLoginDeps(t, mc, func(cfg *dongxi.Config) error {
		savedCfg = cfg
		return nil
	})

	_, err := runLoginWith(t, "a@b.com", "secret")
	if err != nil {
		t.Fatalf("runLogin returned error: %v", err)
	}
	if savedCfg == nil {
		t.Fatal("saveConfigFunc was not called")
	}
	if savedCfg.Email != "a@b.com" {
		t.Errorf("saved email = %q, want %q", savedCfg.Email, "a@b.com")
	}
	if savedCfg.Password != "secret" {
		t.Errorf("saved password = %q, want %q", savedCfg.Password, "secret")
	}
	if savedCfg.HistoryKey != "hk-123" {
		t.Errorf("saved history key = %q, want %q", savedCfg.HistoryKey, "hk-123")
	}
}

func TestRunLoginVerifyFails(t *testing.T) {
	mc := &loginMockClient{
		err: errors.New("bad credentials"),
	}
	overrideLoginDeps(t, mc, func(cfg *dongxi.Config) error {
		t.Fatal("saveConfigFunc should not be called when verify fails")
		return nil
	})

	_, err := runLoginWith(t, "a@b.com", "wrong")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "verify credentials") {
		t.Errorf("error = %q, want it to contain 'verify credentials'", err.Error())
	}
}

func TestRunLoginSaveConfigFails(t *testing.T) {
	mc := &loginMockClient{
		account: &dongxi.Account{HistoryKey: "hk-123", Email: "a@b.com", Status: "active"},
	}
	overrideLoginDeps(t, mc, func(cfg *dongxi.Config) error {
		return errors.New("disk full")
	})

	_, err := runLoginWith(t, "a@b.com", "secret")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "save config") {
		t.Errorf("error = %q, want it to contain 'save config'", err.Error())
	}
}

func TestRunLoginOutput(t *testing.T) {
	mc := &loginMockClient{
		account: &dongxi.Account{HistoryKey: "hk-456", Email: "test@example.com", Status: "active"},
	}
	overrideLoginDeps(t, mc, func(cfg *dongxi.Config) error { return nil })

	buf, err := runLoginWith(t, "test@example.com", "pw")
	if err != nil {
		t.Fatalf("runLogin returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "test@example.com") {
		t.Errorf("output missing email, got: %q", out)
	}
	if !strings.Contains(out, "hk-456") {
		t.Errorf("output missing history key, got: %q", out)
	}
}
