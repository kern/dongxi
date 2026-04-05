package cmd

import (
	"bytes"
	"fmt"
	"os"
	"testing"
)

func TestRunInfo(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runInfo(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()
	if !bytes.Contains([]byte(out), []byte("DONGXI INFO")) {
		t.Error("expected info header in output")
	}
	if !bytes.Contains([]byte(out), []byte("Server Index")) {
		t.Error("expected server index in output")
	}
}

func TestRunInfoJSON(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	old := flagJSON
	flagJSON = true
	defer func() { flagJSON = old }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runInfo(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !bytes.Contains(buf.Bytes(), []byte("server_index")) {
		t.Error("expected server_index in JSON output")
	}
}

func TestRunInfoLoadStateErr(t *testing.T) {
	setupMockStateErr(t, fmt.Errorf("mock load error"))
	err := runInfo(nil, nil)
	if err == nil || err.Error() != "mock load error" {
		t.Fatalf("expected load error, got %v", err)
	}
}

func TestRunInfoGetAccountErr(t *testing.T) {
	mock := setupMockState(t, nil)
	mock.getAccountErr = fmt.Errorf("account error")
	err := runInfo(nil, nil)
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("account error")) {
		t.Fatalf("expected account error, got %v", err)
	}
}

func TestRunInfoGetHistoryErr(t *testing.T) {
	mock := setupMockState(t, nil)
	mock.getHistoryErr = fmt.Errorf("history error")
	err := runInfo(nil, nil)
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("history error")) {
		t.Fatalf("expected history error, got %v", err)
	}
}
