package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"expenses/bot/internal/api"
	"expenses/bot/internal/config"
	"expenses/bot/internal/llm"
	"expenses/bot/internal/repository/postgres"
	"expenses/bot/internal/telegram"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {

	cfg, err := config.NewConfig(
		os.Getenv("DATABASE_URL"),
		os.Getenv("MIGRATIONS_DIR"),
		os.Getenv("TELEGRAM_BOT_TOKEN"),
		os.Getenv("WEBHOOK_PUBLIC_URL"),
		os.Getenv("WEBHOOK_LISTEN"),
		os.Getenv("WEBHOOK_SECRET"),
		os.Getenv("API_LISTEN"),
		os.Getenv("LLM_API_KEY"),
		os.Getenv("LLM_BASE_URL"),
		os.Getenv("LLM_MODEL"),
	)
	if err != nil {
		log.Fatal(err)
	}

	if err := postgres.Migrate(cfg.DatabaseURL, cfg.MigrationsDir); err != nil {
		log.Fatal(err)
	}
	fmt.Println("migrations: ok")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatal(err)
	}
	repo := postgres.NewExpensesRepo(pool)
	fmt.Println("postgres: ok")

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	llmClient, err := llm.NewClient(cfg.LLMAPIKey, cfg.LLMBaseURL, cfg.LLMModel, repo)
	if err != nil {
		log.Fatal(err)
	}

	bot, err := telegram.New(cfg.TelegramToken, telegram.WebhookConfig{
		PublicURL: cfg.WebhookPublicURL,
		Listen:    cfg.WebhookListen,
		Secret:    cfg.WebhookSecret,
	}, llmClient.Handler, logger)
	if err != nil {
		log.Fatal(err)
	}

	apiServer := api.NewServer(cfg.APIListen, repo, logger)

	go func() {
		if err := apiServer.Start(); err != nil {
			log.Fatal(err)
		}
	}()

	go func() {
		sigCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()
		<-sigCtx.Done()
		logger.Info("shutting down")
		bot.Stop()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		_ = apiServer.Shutdown(shutdownCtx)
	}()

	bot.Start()
}
