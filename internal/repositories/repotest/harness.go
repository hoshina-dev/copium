//go:build integration

// Package repotest hosts a Postgres testcontainers harness shared by repo tests.
package repotest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	once      sync.Once
	shared    *gorm.DB
	sharedErr error
)

// DB returns a singleton *gorm.DB connected to a freshly-migrated Postgres
// running in a testcontainer. Skips the test if Docker isn't available.
func DB(t *testing.T) *gorm.DB {
	t.Helper()
	once.Do(func() {
		shared, sharedErr = bootstrap()
	})
	if sharedErr != nil {
		t.Skipf("docker/postgres unavailable: %v", sharedErr)
	}
	t.Cleanup(func() { wipe(t, shared) })
	return shared
}

func bootstrap() (*gorm.DB, error) {
	ctx := context.Background()
	c, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("copium"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		return nil, err
	}
	dsn, err := c.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, err
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err := applyMigrations(db); err != nil {
		return nil, err
	}
	return db, nil
}

func applyMigrations(db *gorm.DB) error {
	root, err := repoRoot()
	if err != nil {
		return err
	}
	for _, name := range []string{"001_init.up.sql"} {
		path := filepath.Join(root, "sql", name)
		b, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		if err := db.Exec(string(b)).Error; err != nil {
			return fmt.Errorf("apply %s: %w", name, err)
		}
	}
	return nil
}

// repoRoot finds the project root by walking up from this source file until
// it sees a sql/ directory.
func repoRoot() (string, error) {
	_, file, _, _ := runtime.Caller(0)
	d := filepath.Dir(file)
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(d, "sql", "001_init.up.sql")); err == nil {
			return d, nil
		}
		d = filepath.Dir(d)
	}
	return "", fmt.Errorf("could not find repo root")
}

func wipe(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Exec(`TRUNCATE email_outbox, email_template_versions, email_templates RESTART IDENTITY CASCADE`).Error; err != nil {
		t.Fatalf("truncate: %v", err)
	}
}
