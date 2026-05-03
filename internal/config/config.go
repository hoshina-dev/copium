// Package config loads runtime configuration from environment variables.
//
// Load takes a getenv func (typically os.Getenv) so tests can pass stubs
// without mutating real env. The returned Config is fully validated.
package config

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var (
	ErrMissingRequired = errors.New("missing required config")
	ErrInvalidValue    = errors.New("invalid config value")
)

type GetenvFunc func(string) string

type Config struct {
	Port           int
	DataSourceName string

	Custapi CustapiConfig
	Sender  SenderConfig
	Worker  WorkerConfig
	Otel    OtelConfig
}

type CustapiConfig struct {
	BaseURL string
	Timeout time.Duration
}

type SenderConfig struct {
	Provider    string // noop | smtp | ses | sendgrid
	DefaultFrom string
	SMTP        SMTPConfig
}

type SMTPConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	UseTLS   bool
}

type WorkerConfig struct {
	Enabled      bool
	PollInterval time.Duration
	BatchSize    int
	MaxAttempts  int
}

type OtelConfig struct {
	Enabled         bool
	ServiceName     string
	OTLPEndpoint    string
	OTLPInsecure    bool
}

var validProviders = map[string]struct{}{
	"noop": {}, "smtp": {}, "ses": {}, "sendgrid": {},
}

func Load(getenv GetenvFunc) (*Config, error) {
	cfg := &Config{
		Port: getInt(getenv, "PORT", 8081),
		Custapi: CustapiConfig{
			BaseURL: getenv("CUSTAPI_BASE_URL"),
			Timeout: time.Duration(getInt(getenv, "CUSTAPI_TIMEOUT_MS", 3000)) * time.Millisecond,
		},
		Sender: SenderConfig{
			Provider:    getStr(getenv, "EMAIL_PROVIDER", "noop"),
			DefaultFrom: getenv("EMAIL_DEFAULT_FROM"),
			SMTP: SMTPConfig{
				Host:     getenv("SMTP_HOST"),
				Port:     getInt(getenv, "SMTP_PORT", 25),
				User:     getenv("SMTP_USER"),
				Password: getenv("SMTP_PASSWORD"),
				UseTLS:   getBool(getenv, "SMTP_TLS", false),
			},
		},
		Worker: WorkerConfig{
			Enabled:      getBool(getenv, "WORKER_ENABLED", true),
			PollInterval: time.Duration(getInt(getenv, "WORKER_POLL_INTERVAL_MS", 2000)) * time.Millisecond,
			BatchSize:    getInt(getenv, "WORKER_BATCH_SIZE", 10),
			MaxAttempts:  getInt(getenv, "WORKER_MAX_ATTEMPTS", 5),
		},
		Otel: OtelConfig{
			Enabled:      getBool(getenv, "OTEL_ENABLED", false),
			ServiceName:  getStr(getenv, "OTEL_SERVICE_NAME", "copium"),
			OTLPEndpoint: getStr(getenv, "OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317"),
			OTLPInsecure: getBool(getenv, "OTEL_EXPORTER_OTLP_INSECURE", true),
		},
	}

	cfg.DataSourceName = buildDSN(getenv)

	if err := validate(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func validate(c *Config) error {
	if c.Custapi.BaseURL == "" {
		return fmt.Errorf("%w: CUSTAPI_BASE_URL", ErrMissingRequired)
	}
	if c.DataSourceName == "" {
		return fmt.Errorf("%w: DATA_SOURCE_NAME or DB_HOST/DB_USER/DB_NAME", ErrMissingRequired)
	}
	if c.Sender.DefaultFrom == "" {
		return fmt.Errorf("%w: EMAIL_DEFAULT_FROM", ErrMissingRequired)
	}
	if _, ok := validProviders[c.Sender.Provider]; !ok {
		return fmt.Errorf("%w: EMAIL_PROVIDER=%q (want one of noop|smtp|ses|sendgrid)",
			ErrInvalidValue, c.Sender.Provider)
	}
	if c.Sender.Provider == "smtp" && c.Sender.SMTP.Host == "" {
		return fmt.Errorf("%w: SMTP_HOST (required when EMAIL_PROVIDER=smtp)", ErrMissingRequired)
	}
	return nil
}

func buildDSN(getenv GetenvFunc) string {
	if direct := getenv("DATA_SOURCE_NAME"); direct != "" {
		return direct
	}
	host := getenv("DB_HOST")
	user := getenv("DB_USER")
	name := getenv("DB_NAME")
	if host == "" || user == "" || name == "" {
		return ""
	}
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		host, user, getenv("DB_PASSWORD"), name,
		getStr(getenv, "DB_PORT", "5432"),
		getStr(getenv, "DB_SSLMODE", "disable"))
}

func getStr(getenv GetenvFunc, key, def string) string {
	if v := getenv(key); v != "" {
		return v
	}
	return def
}

func getInt(getenv GetenvFunc, key string, def int) int {
	v := getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func getBool(getenv GetenvFunc, key string, def bool) bool {
	v := strings.ToLower(strings.TrimSpace(getenv(key)))
	switch v {
	case "":
		return def
	case "1", "t", "true", "y", "yes", "on":
		return true
	case "0", "f", "false", "n", "no", "off":
		return false
	default:
		return def
	}
}
