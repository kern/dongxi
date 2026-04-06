package dongxi

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveAndLoadConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg := &Config{
		Email:      "test@example.com",
		Password:   "secret123",
		HistoryKey: "abc-def-ghi",
	}

	if err := SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}

	// Verify file permissions.
	path := filepath.Join(home, ".config", "dongxi", "config.json")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("file permissions = %o, want 0600", perm)
	}

	// Verify directory permissions.
	dirInfo, err := os.Stat(filepath.Join(home, ".config", "dongxi"))
	if err != nil {
		t.Fatal(err)
	}
	if perm := dirInfo.Mode().Perm(); perm != 0700 {
		t.Errorf("directory permissions = %o, want 0700", perm)
	}

	// Load and compare.
	loaded, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Email != cfg.Email {
		t.Errorf("Email = %q, want %q", loaded.Email, cfg.Email)
	}
	if loaded.Password != cfg.Password {
		t.Errorf("Password = %q, want %q", loaded.Password, cfg.Password)
	}
	if loaded.HistoryKey != cfg.HistoryKey {
		t.Errorf("HistoryKey = %q, want %q", loaded.HistoryKey, cfg.HistoryKey)
	}
}

func TestLoadConfigMissing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error for missing config")
	}
	if got := err.Error(); got != "not logged in — run 'dongxi login' first" {
		t.Errorf("error = %q, want 'not logged in' message", got)
	}
}

func TestLoadConfigInvalidJSON(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	dir := filepath.Join(home, ".config", "dongxi")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte("{invalid"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestConfigDirAndPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	dir, err := ConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	if dir != filepath.Join(home, ".config", "dongxi") {
		t.Errorf("ConfigDir() = %q, want %q", dir, filepath.Join(home, ".config", "dongxi"))
	}

	path, err := ConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	if path != filepath.Join(home, ".config", "dongxi", "config.json") {
		t.Errorf("ConfigPath() = %q, want %q", path, filepath.Join(home, ".config", "dongxi", "config.json"))
	}
}

// --- ConfigDir error path (HOME unset) ---

func TestConfigDirNoHome(t *testing.T) {
	t.Setenv("HOME", "")

	_, err := ConfigDir()
	if err == nil {
		t.Fatal("expected error when HOME is empty")
	}
	if !strings.Contains(err.Error(), "find home directory") {
		t.Errorf("error = %q, want 'find home directory'", err.Error())
	}
}

// --- ConfigPath error propagation ---

func TestConfigPathNoHome(t *testing.T) {
	t.Setenv("HOME", "")

	_, err := ConfigPath()
	if err == nil {
		t.Fatal("expected error when HOME is empty")
	}
}

// --- LoadConfig error when HOME is unset ---

func TestLoadConfigNoHome(t *testing.T) {
	t.Setenv("HOME", "")

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error when HOME is empty")
	}
}

// --- LoadConfig read error (permission denied, not IsNotExist) ---

func TestLoadConfigPermissionError(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	dir := filepath.Join(home, ".config", "dongxi")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{"email":"x"}`), 0600); err != nil {
		t.Fatal(err)
	}
	// Remove read permission.
	if err := os.Chmod(path, 0000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(path, 0600) })

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error for unreadable config")
	}
	if !strings.Contains(err.Error(), "read config") {
		t.Errorf("error = %q, want 'read config'", err.Error())
	}
}

// --- SaveConfig error when HOME is unset ---

func TestSaveConfigNoHome(t *testing.T) {
	t.Setenv("HOME", "")

	err := SaveConfig(&Config{Email: "x"})
	if err == nil {
		t.Fatal("expected error when HOME is empty")
	}
}

// --- SaveConfig MkdirAll error ---

func TestSaveConfigMkdirError(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Create a file where the directory should be, so MkdirAll fails.
	configParent := filepath.Join(home, ".config")
	if err := os.MkdirAll(configParent, 0700); err != nil {
		t.Fatal(err)
	}
	// Place a regular file at the dongxi path so MkdirAll fails.
	if err := os.WriteFile(filepath.Join(configParent, "dongxi"), []byte("blocker"), 0600); err != nil {
		t.Fatal(err)
	}

	err := SaveConfig(&Config{Email: "x"})
	if err == nil {
		t.Fatal("expected error for MkdirAll failure")
	}
	if !strings.Contains(err.Error(), "create config directory") {
		t.Errorf("error = %q, want 'create config directory'", err.Error())
	}
}

// --- SaveConfig WriteFile error ---

func TestSaveConfigWriteError(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	dir := filepath.Join(home, ".config", "dongxi")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	// Make directory read-only so WriteFile fails.
	if err := os.Chmod(dir, 0500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0700) })

	err := SaveConfig(&Config{Email: "x"})
	if err == nil {
		t.Fatal("expected error for write failure")
	}
	if !strings.Contains(err.Error(), "write config") {
		t.Errorf("error = %q, want 'write config'", err.Error())
	}
}
