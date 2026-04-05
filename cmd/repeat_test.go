package cmd

import (
	"bytes"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/kern/dongxi/dongxi"
	"github.com/spf13/cobra"
)

func TestParseRepeatFrequencyDaily(t *testing.T) {
	rr, err := parseRepeatFrequency("1 daily")
	if err != nil {
		t.Fatal(err)
	}
	if toInt(rr[dongxi.RepeatFreqUnit]) != dongxi.FreqUnitDaily {
		t.Errorf("fu = %v, want %d (daily)", rr[dongxi.RepeatFreqUnit], dongxi.FreqUnitDaily)
	}
	if toInt(rr[dongxi.RepeatFreqAmount]) != 1 {
		t.Errorf("fa = %v, want 1", rr[dongxi.RepeatFreqAmount])
	}
	if toInt(rr[dongxi.RepeatVersion]) != 4 {
		t.Errorf("rrv = %v, want 4", rr[dongxi.RepeatVersion])
	}
	if toInt(rr[dongxi.RepeatType]) != dongxi.RepeatFixedSchedule {
		t.Errorf("tp = %v, want %d (fixed)", rr[dongxi.RepeatType], dongxi.RepeatFixedSchedule)
	}
}

func TestParseRepeatFrequencyWeekly(t *testing.T) {
	rr, err := parseRepeatFrequency("2 weekly")
	if err != nil {
		t.Fatal(err)
	}
	if toInt(rr[dongxi.RepeatFreqUnit]) != dongxi.FreqUnitWeekly {
		t.Errorf("fu = %v, want %d (weekly)", rr[dongxi.RepeatFreqUnit], dongxi.FreqUnitWeekly)
	}
	if toInt(rr[dongxi.RepeatFreqAmount]) != 2 {
		t.Errorf("fa = %v, want 2", rr[dongxi.RepeatFreqAmount])
	}
	// Check offset has wd matching current weekday.
	offsets, ok := rr[dongxi.RepeatOffset].([]any)
	if !ok || len(offsets) != 1 {
		t.Fatalf("of = %v, want 1-element array", rr[dongxi.RepeatOffset])
	}
	offset, ok := offsets[0].(map[string]any)
	if !ok {
		t.Fatalf("of[0] = %T, want map[string]any", offsets[0])
	}
	if toInt(offset[dongxi.OffsetWeekday]) != int(time.Now().Weekday()) {
		t.Errorf("wd = %v, want %d (today's weekday)", offset[dongxi.OffsetWeekday], time.Now().Weekday())
	}
}

func TestParseRepeatFrequencyMonthly(t *testing.T) {
	rr, err := parseRepeatFrequency("3 monthly")
	if err != nil {
		t.Fatal(err)
	}
	if toInt(rr[dongxi.RepeatFreqUnit]) != dongxi.FreqUnitMonthly {
		t.Errorf("fu = %v, want %d (monthly)", rr[dongxi.RepeatFreqUnit], dongxi.FreqUnitMonthly)
	}
	if toInt(rr[dongxi.RepeatFreqAmount]) != 3 {
		t.Errorf("fa = %v, want 3", rr[dongxi.RepeatFreqAmount])
	}
	offsets, ok := rr[dongxi.RepeatOffset].([]any)
	if !ok || len(offsets) != 1 {
		t.Fatalf("of = %v, want 1-element array", rr[dongxi.RepeatOffset])
	}
	offset, ok := offsets[0].(map[string]any)
	if !ok {
		t.Fatalf("of[0] = %T, want map[string]any", offsets[0])
	}
	if toInt(offset[dongxi.OffsetDay]) != time.Now().Day() {
		t.Errorf("dy = %v, want %d (today's day)", offset[dongxi.OffsetDay], time.Now().Day())
	}
}

func TestParseRepeatFrequencyAltUnits(t *testing.T) {
	alts := []struct {
		input string
		fu    int
	}{
		{"1 day", dongxi.FreqUnitDaily},
		{"1 days", dongxi.FreqUnitDaily},
		{"1 week", dongxi.FreqUnitWeekly},
		{"1 weeks", dongxi.FreqUnitWeekly},
		{"1 month", dongxi.FreqUnitMonthly},
		{"1 months", dongxi.FreqUnitMonthly},
	}
	for _, tt := range alts {
		t.Run(tt.input, func(t *testing.T) {
			rr, err := parseRepeatFrequency(tt.input)
			if err != nil {
				t.Fatal(err)
			}
			if toInt(rr[dongxi.RepeatFreqUnit]) != tt.fu {
				t.Errorf("fu = %v, want %d", rr[dongxi.RepeatFreqUnit], tt.fu)
			}
		})
	}
}

func TestParseRepeatFrequencyEndDate(t *testing.T) {
	rr, err := parseRepeatFrequency("1 daily")
	if err != nil {
		t.Fatal(err)
	}
	if toInt(rr[dongxi.RepeatEndDate]) != int(dongxi.RepeatEndNever) {
		t.Errorf("ed = %v, want %d (far future)", rr[dongxi.RepeatEndDate], dongxi.RepeatEndNever)
	}
	if toInt(rr[dongxi.RepeatCount]) != 0 {
		t.Errorf("rc = %v, want 0 (unlimited)", rr[dongxi.RepeatCount])
	}
}

func TestParseRepeatFrequencyWeeklyWithDays(t *testing.T) {
	// Set the global flag to simulate --days flag.
	old := flagRepeatDays
	flagRepeatDays = "mon,wed,fri"
	defer func() { flagRepeatDays = old }()

	rr, err := parseRepeatFrequency("1 weekly")
	if err != nil {
		t.Fatal(err)
	}
	offsets, ok := rr[dongxi.RepeatOffset].([]any)
	if !ok {
		t.Fatalf("of = %T, want []any", rr[dongxi.RepeatOffset])
	}
	if len(offsets) != 3 {
		t.Fatalf("of len = %d, want 3", len(offsets))
	}
	// Check that it parsed Monday(1), Wednesday(3), Friday(5).
	wds := map[int]bool{}
	for _, o := range offsets {
		m := o.(map[string]any)
		wds[toInt(m[dongxi.OffsetWeekday])] = true
	}
	if !wds[1] || !wds[3] || !wds[5] {
		t.Errorf("weekdays = %v, want {1,3,5}", wds)
	}
}

func TestParseRepeatFrequencyBadWeekday(t *testing.T) {
	old := flagRepeatDays
	flagRepeatDays = "mon,xyz"
	defer func() { flagRepeatDays = old }()

	_, err := parseRepeatFrequency("1 weekly")
	if err == nil {
		t.Error("expected error for bad weekday")
	}
}

func TestParseRepeatFrequencyAfterCompletion(t *testing.T) {
	old := flagRepeatType
	flagRepeatType = "completion"
	defer func() { flagRepeatType = old }()

	rr, err := parseRepeatFrequency("1 daily")
	if err != nil {
		t.Fatal(err)
	}
	if toInt(rr[dongxi.RepeatType]) != dongxi.RepeatAfterCompletion {
		t.Errorf("tp = %v, want %d (after completion)", rr[dongxi.RepeatType], dongxi.RepeatAfterCompletion)
	}
}

func TestParseRepeatFrequencyWithEndDate(t *testing.T) {
	old := flagRepeatEndDate
	flagRepeatEndDate = "2026-12-31"
	defer func() { flagRepeatEndDate = old }()

	rr, err := parseRepeatFrequency("1 daily")
	if err != nil {
		t.Fatal(err)
	}
	ed := toInt(rr[dongxi.RepeatEndDate])
	if ed == int(dongxi.RepeatEndNever) {
		t.Errorf("end date should not be far future sentinel")
	}
	if ed <= 0 {
		t.Errorf("end date = %d, want positive timestamp", ed)
	}
}

func TestParseRepeatFrequencyWithBadEndDate(t *testing.T) {
	old := flagRepeatEndDate
	flagRepeatEndDate = "not-a-date"
	defer func() { flagRepeatEndDate = old }()

	_, err := parseRepeatFrequency("1 daily")
	if err == nil {
		t.Error("expected error for bad end date")
	}
}

func TestParseRepeatFrequencyWithEndCount(t *testing.T) {
	old := flagRepeatEndCount
	flagRepeatEndCount = 10
	defer func() { flagRepeatEndCount = old }()

	rr, err := parseRepeatFrequency("1 weekly")
	if err != nil {
		t.Fatal(err)
	}
	if toInt(rr[dongxi.RepeatCount]) != 10 {
		t.Errorf("rc = %v, want 10", rr[dongxi.RepeatCount])
	}
}

func TestParseRepeatFrequencyErrors(t *testing.T) {
	bad := []struct {
		name  string
		input string
	}{
		{"zero amount", "0 daily"},
		{"negative", "-1 weekly"},
		{"non-integer", "abc daily"},
		{"unknown unit", "1 yearly"},
		{"missing amount", "daily"},
		{"too many parts", "1 2 daily"},
		{"empty", ""},
	}
	for _, tt := range bad {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseRepeatFrequency(tt.input)
			if err == nil {
				t.Errorf("parseRepeatFrequency(%q) should have failed", tt.input)
			}
		})
	}
}

// makeRepeatCmd creates a cobra.Command with the same flags as repeatCmd.
func makeRepeatCmd() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Flags().StringVar(&flagRepeatFrequency, "every", "", "")
	cmd.Flags().BoolVar(&flagRepeatClear, "clear", false, "")
	cmd.Flags().StringVar(&flagRepeatType, "type", "fixed", "")
	cmd.Flags().StringVar(&flagRepeatDays, "days", "", "")
	cmd.Flags().StringVar(&flagRepeatEndDate, "end-date", "", "")
	cmd.Flags().IntVar(&flagRepeatEndCount, "end-count", 0, "")
	return cmd
}

func TestRunRepeatSetDaily(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	cmd := makeRepeatCmd()
	flagRepeatFrequency = "1 daily"
	flagRepeatClear = false

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runRepeat(cmd, []string{"task-1"})

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["task-1"]
	if commit.P[dongxi.FieldRepeatRule] == nil {
		t.Error("expected repeat rule to be set")
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !bytes.Contains(buf.Bytes(), []byte("1 daily")) {
		t.Error("expected frequency in output")
	}
}

func TestRunRepeatClear(t *testing.T) {
	mock := setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	cmd := makeRepeatCmd()
	flagRepeatFrequency = ""
	flagRepeatClear = true

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runRepeat(cmd, []string{"task-1"})

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	commit := mock.lastCommit["task-1"]
	if _, ok := commit.P[dongxi.FieldRepeatRule]; !ok {
		t.Error("expected repeat rule key in payload")
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !bytes.Contains(buf.Bytes(), []byte("cleared repeat")) {
		t.Error("expected 'cleared repeat' in output")
	}
}

func TestRunRepeatNoFlags(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	cmd := makeRepeatCmd()
	flagRepeatFrequency = ""
	flagRepeatClear = false

	err := runRepeat(cmd, []string{"task-1"})
	if err == nil {
		t.Fatal("expected error when neither --every nor --clear given")
	}
}

func TestRunRepeatNotATask(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeArea("area-1", "Work"),
	})

	cmd := makeRepeatCmd()
	flagRepeatFrequency = "1 daily"
	flagRepeatClear = false

	err := runRepeat(cmd, []string{"area-1"})
	if err == nil {
		t.Fatal("expected error for non-task entity")
	}
}

func TestRunRepeatNotFound(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	cmd := makeRepeatCmd()
	flagRepeatFrequency = "1 daily"
	flagRepeatClear = false

	err := runRepeat(cmd, []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent UUID")
	}
}

func TestRunRepeatBadFrequency(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	cmd := makeRepeatCmd()
	flagRepeatFrequency = "bad"
	flagRepeatClear = false

	err := runRepeat(cmd, []string{"task-1"})
	if err == nil {
		t.Fatal("expected error for bad frequency")
	}
}

func TestRunRepeatJSON(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	old := flagJSON
	flagJSON = true
	defer func() { flagJSON = old }()

	cmd := makeRepeatCmd()
	flagRepeatFrequency = "1 daily"
	flagRepeatClear = false

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runRepeat(cmd, []string{"task-1"})

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if buf.Len() == 0 {
		t.Error("expected JSON output")
	}
}

func TestRunRepeatJSONClear(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", "Buy milk"),
	})

	old := flagJSON
	flagJSON = true
	defer func() { flagJSON = old }()

	cmd := makeRepeatCmd()
	flagRepeatFrequency = ""
	flagRepeatClear = true

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runRepeat(cmd, []string{"task-1"})

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !bytes.Contains(buf.Bytes(), []byte("cleared repeat")) {
		t.Error("expected 'cleared repeat' in JSON output")
	}
}

func TestRunRepeatUntitledTask(t *testing.T) {
	setupMockState(t, []map[string]any{
		makeTask("task-1", ""),
	})

	cmd := makeRepeatCmd()
	flagRepeatFrequency = "1 daily"
	flagRepeatClear = false

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runRepeat(cmd, []string{"task-1"})

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !bytes.Contains(buf.Bytes(), []byte("(untitled)")) {
		t.Error("expected (untitled) in output")
	}
}

func TestRunRepeatLoadStateErr(t *testing.T) {
	setupMockStateErr(t, fmt.Errorf("mock load error"))
	cmd := makeRepeatCmd()
	flagRepeatFrequency = "1 daily"
	flagRepeatClear = false
	err := runRepeat(cmd, []string{"task-1"})
	if err == nil || err.Error() != "mock load error" {
		t.Fatalf("expected load error, got %v", err)
	}
}

func TestRunRepeatGetHistoryErr(t *testing.T) {
	mock := setupMockState(t, []map[string]any{makeTask("task-1", "Buy milk")})
	mock.getHistoryErr = fmt.Errorf("history error")
	cmd := makeRepeatCmd()
	flagRepeatFrequency = "1 daily"
	flagRepeatClear = false
	err := runRepeat(cmd, []string{"task-1"})
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("history error")) {
		t.Fatalf("expected history error, got %v", err)
	}
}

func TestRunRepeatCommitErr(t *testing.T) {
	mock := setupMockState(t, []map[string]any{makeTask("task-1", "Buy milk")})
	mock.commitErr = fmt.Errorf("commit error")
	cmd := makeRepeatCmd()
	flagRepeatFrequency = "1 daily"
	flagRepeatClear = false
	err := runRepeat(cmd, []string{"task-1"})
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("commit error")) {
		t.Fatalf("expected commit error, got %v", err)
	}
}
