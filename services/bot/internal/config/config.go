package config

import "errors"

type Config struct {
	DatabaseURL   string
	MigrationsDir string
	TelegramToken string

	WebhookPublicURL string
	WebhookListen    string
	WebhookSecret    string

	APIListen string

	LLMAPIKey  string
	LLMBaseURL string
	LLMModel   string
}

var ErrConfig = errors.New("err config")

func NewConfig(
	databaseURL, migrationsDir, telegramToken,
	webhookPublicURL, webhookListen, webhookSecret,
	apiListen,
	llmAPIKey, llmBaseURL, llmModel string,
) (*Config, error) {
	if databaseURL == "" || migrationsDir == "" || telegramToken == "" {
		return nil, ErrConfig
	}
	if webhookPublicURL == "" || webhookListen == "" {
		return nil, ErrConfig
	}
	if apiListen == "" {
		return nil, ErrConfig
	}
	if llmAPIKey == "" || llmModel == "" {
		return nil, ErrConfig
	}
	if llmBaseURL == "" {
		llmBaseURL = "https://openrouter.ai/api/v1"
	}
	return &Config{
		DatabaseURL:      databaseURL,
		MigrationsDir:    migrationsDir,
		TelegramToken:    telegramToken,
		WebhookPublicURL: webhookPublicURL,
		WebhookListen:    webhookListen,
		WebhookSecret:    webhookSecret,
		APIListen:        apiListen,
		LLMAPIKey:        llmAPIKey,
		LLMBaseURL:       llmBaseURL,
		LLMModel:         llmModel,
	}, nil
}
