package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"expenses/bot/internal/repository"

	openai "github.com/sashabaranov/go-openai"
)

type Client struct {
	api   *openai.Client
	model string
	repo  repository.ExpenseRepo
	loc   *time.Location
}

func NewClient(apiKey, baseURL, model string, repo repository.ExpenseRepo) (*Client, error) {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return nil, fmt.Errorf("timezone: %w", err)
	}
	cfg := openai.DefaultConfig(apiKey)
	if baseURL != "" {
		cfg.BaseURL = baseURL
	}
	return &Client{
		api:   openai.NewClientWithConfig(cfg),
		model: model,
		repo:  repo,
		loc:   loc,
	}, nil
}

func (c *Client) Reply(ctx context.Context, userID int64, userText string) (string, error) {
	now := time.Now().In(c.loc)
	messages := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
		{
			Role: openai.ChatMessageRoleUser,
			Content: fmt.Sprintf(
				"Сейчас: %s (Europe/Moscow).\nСообщение: %s",
				now.Format("2006-01-02 15:04"),
				userText,
			),
		},
	}

	for range 4 {
		resp, err := c.api.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
			Model:       c.model,
			Messages:    messages,
			Tools:       tools(),
			ToolChoice:  "auto",
			Temperature: 0.2,
		})
		if err != nil {
			return "", fmt.Errorf("llm: %w", err)
		}
		if len(resp.Choices) == 0 {
			return "", fmt.Errorf("llm: empty response")
		}

		msg := resp.Choices[0].Message
		messages = append(messages, msg)

		if len(msg.ToolCalls) == 0 {
			text := strings.TrimSpace(msg.Content)
			if text == "" {
				return "Готово.", nil
			}
			return text, nil
		}

		for _, call := range msg.ToolCalls {
			result, callErr := c.executeTool(ctx, userID, call, now)
			content := result
			if callErr != nil {
				b, _ := json.Marshal(map[string]string{"error": callErr.Error()})
				content = string(b)
			}
			messages = append(messages, openai.ChatCompletionMessage{
				Role:       openai.ChatMessageRoleTool,
				ToolCallID: call.ID,
				Content:    content,
			})
		}
	}

	return "Не удалось завершить запрос. Уточните, пожалуйста.", nil
}

func (c *Client) Handler(userID int64, text string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	return c.Reply(ctx, userID, text)
}
