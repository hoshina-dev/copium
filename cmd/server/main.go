// Command server is the copium HTTP server.
//
// This file is the COMPOSITION ROOT - the only place that imports concrete
// adapter packages (sender, clients/custapi, repositories, clock, idgen).
// Everything below the services layer receives its collaborators through
// constructor injection.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/contrib/otelfiber/v2"
	"github.com/joho/godotenv"

	"github.com/hoshina-dev/copium/internal/clients/custapi"
	"github.com/hoshina-dev/copium/internal/clock"
	"github.com/hoshina-dev/copium/internal/config"
	"github.com/hoshina-dev/copium/internal/database"
	"github.com/hoshina-dev/copium/internal/handlers"
	"github.com/hoshina-dev/copium/internal/idgen"
	"github.com/hoshina-dev/copium/internal/observability"
	"github.com/hoshina-dev/copium/internal/renderer"
	"github.com/hoshina-dev/copium/internal/repositories"
	"github.com/hoshina-dev/copium/internal/routes"
	"github.com/hoshina-dev/copium/internal/sender"
	"github.com/hoshina-dev/copium/internal/services"
	"github.com/hoshina-dev/copium/internal/worker"
)

func main() {
	// .env is optional; ignore the "file not found" case.
	_ = godotenv.Load()

	cfg, err := config.Load(os.Getenv)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	otelP, err := observability.Setup(rootCtx, cfg.Otel)
	if err != nil {
		log.Fatalf("otel: %v", err)
	}
	defer func() {
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = otelP.Shutdown(shutCtx)
	}()

	db, err := database.Connect(cfg.DataSourceName)
	if err != nil {
		log.Fatalf("database: %v", err)
	}

	custClient := custapi.New(cfg.Custapi.BaseURL, custapi.WithTimeout(cfg.Custapi.Timeout))

	snd, err := sender.NewFromConfig(sender.FromConfig{
		Provider: cfg.Sender.Provider,
		SMTP: sender.SMTP{
			Host: cfg.Sender.SMTP.Host, Port: cfg.Sender.SMTP.Port,
			User: cfg.Sender.SMTP.User, Password: cfg.Sender.SMTP.Password,
			UseTLS: cfg.Sender.SMTP.UseTLS,
		},
	})
	if err != nil {
		log.Fatalf("sender: %v", err)
	}
	log.Printf("email sender: %s", snd.Name())

	rdr, _ := renderer.New()
	clk := clock.System{}
	ids := idgen.UUID{}

	tplRepo := repositories.NewTemplateRepo(db)
	verRepo := repositories.NewTemplateVersionRepo(db)
	outRepo := repositories.NewOutboxRepo(db)

	tplSvc := services.NewTemplateService(services.TemplateDeps{
		Templates: tplRepo, TemplateVersions: verRepo, Clock: clk, IDs: ids,
	})
	emSvc := services.NewEmailService(services.EmailDeps{
		Templates: tplRepo, TemplateVersions: verRepo, Outbox: outRepo,
		Users: custClient, Renderer: rdr, Sender: snd,
		Clock: clk, IDs: ids,
		DefaultFrom: cfg.Sender.DefaultFrom,
		MaxAttempts: cfg.Worker.MaxAttempts,
	})

	app := routes.NewApp(routes.Handlers{
		Email:    handlers.NewEmailHandler(emSvc),
		Template: handlers.NewTemplateHandler(tplSvc),
	})
	if cfg.Otel.Enabled {
		app.Use(otelfiber.Middleware())
	}

	if cfg.Worker.Enabled {
		w := worker.New(worker.Deps{
			Store: outRepo, Sender: snd, Clock: clk,
			BatchSize:    cfg.Worker.BatchSize,
			PollInterval: cfg.Worker.PollInterval,
		})
		go w.Run(rootCtx)
		log.Printf("worker enabled (poll=%s, batch=%d)", cfg.Worker.PollInterval, cfg.Worker.BatchSize)
	}

	addr := fmt.Sprintf(":%d", cfg.Port)
	go func() {
		log.Printf("copium listening on %s", addr)
		if err := app.Listen(addr); err != nil {
			log.Printf("server: %v", err)
		}
	}()

	<-rootCtx.Done()
	log.Printf("shutting down...")
	shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = app.ShutdownWithContext(shutCtx)
}
