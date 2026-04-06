package dongxi

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCachePath(t *testing.T) {
	path, err := CachePath()
	if err != nil {
		t.Fatalf("CachePath() error: %v", err)
	}
	if filepath.Base(path) != "history.json" {
		t.Errorf("CachePath() = %q, want basename history.json", path)
	}
}

func TestLoadCacheMissing(t *testing.T) {
	// Point at a temp dir with no cache file.
	tmp := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", tmp)
	defer func() { os.Setenv("HOME", origHome) }()

	c, err := LoadCache()
	if err != nil {
		t.Fatalf("LoadCache() error: %v", err)
	}
	if c.HistoryKey != "" || len(c.Items) != 0 {
		t.Errorf("expected empty cache, got %+v", c)
	}
}

func TestSaveAndLoadCache(t *testing.T) {
	tmp := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", tmp)
	defer func() { os.Setenv("HOME", origHome) }()

	want := &Cache{
		HistoryKey: "hk-abc",
		ItemCount:  2,
		Items: []map[string]any{
			{"uuid1": map[string]any{"t": float64(0)}},
			{"uuid2": map[string]any{"t": float64(1)}},
		},
	}

	if err := SaveCache(want); err != nil {
		t.Fatalf("SaveCache() error: %v", err)
	}

	got, err := LoadCache()
	if err != nil {
		t.Fatalf("LoadCache() error: %v", err)
	}

	if got.HistoryKey != want.HistoryKey {
		t.Errorf("HistoryKey = %q, want %q", got.HistoryKey, want.HistoryKey)
	}
	if got.ItemCount != want.ItemCount {
		t.Errorf("ItemCount = %d, want %d", got.ItemCount, want.ItemCount)
	}
	if len(got.Items) != len(want.Items) {
		t.Errorf("len(Items) = %d, want %d", len(got.Items), len(want.Items))
	}
}

func TestLoadCacheCorrupt(t *testing.T) {
	tmp := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", tmp)
	defer func() { os.Setenv("HOME", origHome) }()

	dir := filepath.Join(tmp, ".config", "dongxi")
	os.MkdirAll(dir, 0700)
	os.WriteFile(filepath.Join(dir, "history.json"), []byte("not json"), 0600)

	c, err := LoadCache()
	if err != nil {
		t.Fatalf("LoadCache() error: %v", err)
	}
	if c.HistoryKey != "" || len(c.Items) != 0 {
		t.Errorf("expected empty cache for corrupt file, got %+v", c)
	}
}

func TestSaveCachePermissions(t *testing.T) {
	tmp := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", tmp)
	defer func() { os.Setenv("HOME", origHome) }()

	if err := SaveCache(&Cache{HistoryKey: "hk"}); err != nil {
		t.Fatalf("SaveCache() error: %v", err)
	}

	path := filepath.Join(tmp, ".config", "dongxi", "history.json")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat cache file: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("cache file permissions = %o, want 0600", perm)
	}
}

func TestLoadCacheReadError(t *testing.T) {
	tmp := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", tmp)
	defer func() { os.Setenv("HOME", origHome) }()

	// Create history.json as a directory so ReadFile fails with a non-NotExist error.
	dir := filepath.Join(tmp, ".config", "dongxi")
	os.MkdirAll(filepath.Join(dir, "history.json"), 0700)

	_, err := LoadCache()
	if err == nil {
		t.Fatal("expected error when cache path is a directory")
	}
}

func TestSaveCacheMarshal(t *testing.T) {
	tmp := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", tmp)
	defer func() { os.Setenv("HOME", origHome) }()

	c := &Cache{HistoryKey: "hk", ItemCount: 1, Items: []map[string]any{{"a": "b"}}}
	if err := SaveCache(c); err != nil {
		t.Fatalf("SaveCache() error: %v", err)
	}

	path := filepath.Join(tmp, ".config", "dongxi", "history.json")
	data, _ := os.ReadFile(path)
	var decoded Cache
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("cache file is not valid JSON: %v", err)
	}
}
