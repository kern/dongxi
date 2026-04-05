package cmd

import (
	"bytes"
	"fmt"
	"os"
	"testing"
)

func TestRunResetWithYes(t *testing.T) {
	setupMockState(t, nil)

	oldFlag := flagResetYes
	flagResetYes = true
	defer func() { flagResetYes = oldFlag }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runReset(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()

	if !bytes.Contains([]byte(out), []byte("test@test.com")) {
		t.Error("expected account email in output")
	}
	if !bytes.Contains([]byte(out), []byte("Resetting history key...")) {
		t.Error("expected 'Resetting history key...' in output")
	}
	if !bytes.Contains([]byte(out), []byte("new-key")) {
		t.Error("expected new history key in output")
	}
	if !bytes.Contains([]byte(out), []byte("History key reset successfully")) {
		t.Error("expected success message in output")
	}
}

func TestRunResetLoadStateErr(t *testing.T) {
	setupMockStateErr(t, fmt.Errorf("mock load error"))
	err := runReset(nil, nil)
	if err == nil || err.Error() != "mock load error" {
		t.Fatalf("expected load error, got %v", err)
	}
}

func TestRunResetGetAccountErr(t *testing.T) {
	mock := setupMockState(t, nil)
	mock.getAccountErr = fmt.Errorf("account error")
	err := runReset(nil, nil)
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("account error")) {
		t.Fatalf("expected account error, got %v", err)
	}
}

func TestRunResetStdinConfirmYes(t *testing.T) {
	setupMockState(t, nil)

	oldFlag := flagResetYes
	flagResetYes = false
	defer func() { flagResetYes = oldFlag }()

	// Mock stdin to type "yes"
	oldStdin := os.Stdin
	stdinR, stdinW, _ := os.Pipe()
	stdinW.WriteString("yes\n")
	stdinW.Close()
	os.Stdin = stdinR
	defer func() { os.Stdin = oldStdin }()

	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := runReset(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}
}

func TestRunResetStdinConfirmNo(t *testing.T) {
	setupMockState(t, nil)

	oldFlag := flagResetYes
	flagResetYes = false
	defer func() { flagResetYes = oldFlag }()

	// Mock stdin to type "no"
	oldStdin := os.Stdin
	stdinR, stdinW, _ := os.Pipe()
	stdinW.WriteString("no\n")
	stdinW.Close()
	os.Stdin = stdinR
	defer func() { os.Stdin = oldStdin }()

	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := runReset(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}
}

func TestRunResetResetHistoryErr(t *testing.T) {
	mock := setupMockState(t, nil)
	mock.resetHistErr = fmt.Errorf("reset error")

	oldFlag := flagResetYes
	flagResetYes = true
	defer func() { flagResetYes = oldFlag }()

	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := runReset(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("reset error")) {
		t.Fatalf("expected reset error, got %v", err)
	}
}
