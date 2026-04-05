package cmd

import (
	"testing"
	"time"
)

func TestToFloat(t *testing.T) {
	tests := []struct {
		input any
		want  float64
	}{
		{float64(3.14), 3.14},
		{float64(0), 0},
		{int(42), 42.0},
		{nil, 0},
		{"3.14", 0},
	}
	for _, tt := range tests {
		got := toFloat(tt.input)
		if got != tt.want {
			t.Errorf("toFloat(%v) = %f, want %f", tt.input, got, tt.want)
		}
	}
}

func TestFormatTS(t *testing.T) {
	// Use a known timestamp.
	ts := float64(1700000000) // 2023-11-14
	got := formatTS(ts)
	expected := time.Unix(1700000000, 0).Local().Format("2006-01-02 15:04")
	if got != expected {
		t.Errorf("formatTS(%f) = %q, want %q", ts, got, expected)
	}
}

func TestFormatTSZero(t *testing.T) {
	got := formatTS(0)
	expected := time.Unix(0, 0).Local().Format("2006-01-02 15:04")
	if got != expected {
		t.Errorf("formatTS(0) = %q, want %q", got, expected)
	}
}
