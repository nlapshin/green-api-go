package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestLoad_DefaultsFromModuleRoot(t *testing.T) {
	t.Chdir(moduleRoot(t))
	t.Setenv("APP_PORT", "8080")
	t.Setenv("GREEN_API_TIMEOUT", "15s")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != 8080 {
		t.Fatalf("port: got %d", cfg.Port)
	}
	if !strings.HasSuffix(cfg.GreenAPIBaseURL, "api.green-api.com") {
		t.Fatalf("base url: %q", cfg.GreenAPIBaseURL)
	}
	if err := cfg.EnsureIndexTemplate(); err != nil {
		t.Fatal(err)
	}
}

func TestLoad_InvalidPort(t *testing.T) {
	t.Chdir(moduleRoot(t))
	t.Setenv("APP_PORT", "not-a-number")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLoad_PORTWhenAPPPortUnset(t *testing.T) {
	t.Chdir(moduleRoot(t))
	t.Setenv("GREEN_API_TIMEOUT", "15s")
	t.Cleanup(func() {
		_ = os.Unsetenv("APP_PORT")
		_ = os.Unsetenv("PORT")
	})
	_ = os.Unsetenv("APP_PORT")
	t.Setenv("PORT", strconv.Itoa(9091))

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != 9091 {
		t.Fatalf("port: got %d want 9091", cfg.Port)
	}
}

func TestLoad_InvalidDuration(t *testing.T) {
	t.Chdir(moduleRoot(t))
	t.Setenv("GREEN_API_TIMEOUT", "forever")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestEnsureIndexTemplate_Missing(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)
	t.Setenv("APP_PORT", "8080")
	t.Setenv("GREEN_API_TIMEOUT", "15s")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.EnsureIndexTemplate(); err == nil || !strings.Contains(err.Error(), "template") {
		t.Fatalf("expected template error, got %v", err)
	}
}

func TestStaticDir_joinsWebRoot(t *testing.T) {
	c := Config{WebRoot: "/app"}
	if got := c.StaticDir(); got != filepath.Join("/app", "web", "static") {
		t.Fatalf("StaticDir: %q", got)
	}
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := wd
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		dir = filepath.Join(dir, "..")
	}
	t.Fatal("go.mod not found")
	return ""
}
