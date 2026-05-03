package config_test

import (
	"errors"
	"testing"
	"time"

	"github.com/hoshina-dev/copium/internal/config"
)

func TestLoad_Defaults(t *testing.T) {
	getenv := stubEnv(map[string]string{
		"CUSTAPI_BASE_URL":      "http://custapi.local",
		"DB_HOST":               "db",
		"DB_PORT":               "5432",
		"DB_USER":               "u",
		"DB_PASSWORD":           "p",
		"DB_NAME":               "copium",
		"DB_SSLMODE":            "disable",
		"EMAIL_DEFAULT_FROM":    "noreply@example.com",
	})
	cfg, err := config.Load(getenv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != 8081 {
		t.Errorf("Port default = %d; want 8081", cfg.Port)
	}
	if cfg.Sender.Provider != "noop" {
		t.Errorf("EMAIL_PROVIDER default = %q; want noop", cfg.Sender.Provider)
	}
	if cfg.Worker.Enabled != true {
		t.Errorf("WORKER_ENABLED default = %v; want true", cfg.Worker.Enabled)
	}
	if cfg.Worker.PollInterval != 2*time.Second {
		t.Errorf("PollInterval default = %v; want 2s", cfg.Worker.PollInterval)
	}
	if cfg.Worker.BatchSize != 10 {
		t.Errorf("BatchSize default = %d; want 10", cfg.Worker.BatchSize)
	}
	if cfg.Worker.MaxAttempts != 5 {
		t.Errorf("MaxAttempts default = %d; want 5", cfg.Worker.MaxAttempts)
	}
	if cfg.Otel.Enabled {
		t.Errorf("OTEL_ENABLED default = true; want false")
	}
	if cfg.Otel.ServiceName != "copium" {
		t.Errorf("ServiceName default = %q; want copium", cfg.Otel.ServiceName)
	}
}

func TestLoad_DSNDirectOverridesParts(t *testing.T) {
	full := "host=h user=u password=p dbname=d port=5432 sslmode=disable"
	getenv := stubEnv(map[string]string{
		"DATA_SOURCE_NAME":   full,
		"DB_HOST":            "ignored",
		"DB_PORT":            "1",
		"DB_USER":            "ignored",
		"DB_PASSWORD":        "ignored",
		"DB_NAME":            "ignored",
		"CUSTAPI_BASE_URL":   "http://x",
		"EMAIL_DEFAULT_FROM": "n@x",
	})
	cfg, err := config.Load(getenv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DataSourceName != full {
		t.Errorf("DSN = %q; want %q", cfg.DataSourceName, full)
	}
}

func TestLoad_DSNBuiltFromParts(t *testing.T) {
	getenv := stubEnv(map[string]string{
		"DB_HOST":            "h",
		"DB_PORT":            "6543",
		"DB_USER":            "alice",
		"DB_PASSWORD":        "s3cret",
		"DB_NAME":            "copium",
		"DB_SSLMODE":         "require",
		"CUSTAPI_BASE_URL":   "http://x",
		"EMAIL_DEFAULT_FROM": "n@x",
	})
	cfg, err := config.Load(getenv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "host=h user=alice password=s3cret dbname=copium port=6543 sslmode=require"
	if cfg.DataSourceName != want {
		t.Errorf("built DSN = %q; want %q", cfg.DataSourceName, want)
	}
}

func TestLoad_OtelEnabledParsesBool(t *testing.T) {
	for _, val := range []string{"true", "TRUE", "1", "yes"} {
		val := val
		t.Run(val, func(t *testing.T) {
			getenv := stubEnv(map[string]string{
				"OTEL_ENABLED":       val,
				"DATA_SOURCE_NAME":   "x",
				"CUSTAPI_BASE_URL":   "http://x",
				"EMAIL_DEFAULT_FROM": "n@x",
			})
			cfg, err := config.Load(getenv)
			if err != nil {
				t.Fatalf("err: %v", err)
			}
			if !cfg.Otel.Enabled {
				t.Errorf("OTEL_ENABLED=%q expected true", val)
			}
		})
	}
}

func TestLoad_MissingCustapiBaseURL(t *testing.T) {
	getenv := stubEnv(map[string]string{
		"DATA_SOURCE_NAME":   "x",
		"EMAIL_DEFAULT_FROM": "n@x",
	})
	_, err := config.Load(getenv)
	if err == nil {
		t.Fatal("expected error for missing CUSTAPI_BASE_URL")
	}
	if !errors.Is(err, config.ErrMissingRequired) {
		t.Errorf("expected ErrMissingRequired, got %v", err)
	}
}

func TestLoad_MissingDB(t *testing.T) {
	getenv := stubEnv(map[string]string{
		"CUSTAPI_BASE_URL":   "http://x",
		"EMAIL_DEFAULT_FROM": "n@x",
	})
	_, err := config.Load(getenv)
	if err == nil {
		t.Fatal("expected error for missing DB config")
	}
	if !errors.Is(err, config.ErrMissingRequired) {
		t.Errorf("expected ErrMissingRequired, got %v", err)
	}
}

func TestLoad_SMTPRequiredWhenProviderSMTP(t *testing.T) {
	getenv := stubEnv(map[string]string{
		"DATA_SOURCE_NAME":   "x",
		"CUSTAPI_BASE_URL":   "http://x",
		"EMAIL_DEFAULT_FROM": "n@x",
		"EMAIL_PROVIDER":     "smtp",
	})
	_, err := config.Load(getenv)
	if err == nil {
		t.Fatal("expected error: SMTP provider needs SMTP_HOST")
	}
	if !errors.Is(err, config.ErrMissingRequired) {
		t.Errorf("expected ErrMissingRequired, got %v", err)
	}
}

func TestLoad_InvalidProvider(t *testing.T) {
	getenv := stubEnv(map[string]string{
		"DATA_SOURCE_NAME":   "x",
		"CUSTAPI_BASE_URL":   "http://x",
		"EMAIL_DEFAULT_FROM": "n@x",
		"EMAIL_PROVIDER":     "carrier-pigeon",
	})
	_, err := config.Load(getenv)
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
	if !errors.Is(err, config.ErrInvalidValue) {
		t.Errorf("expected ErrInvalidValue, got %v", err)
	}
}

func stubEnv(m map[string]string) func(string) string {
	return func(key string) string { return m[key] }
}
