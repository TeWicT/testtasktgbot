package telegram

import (
	"log/slog"

	tele "gopkg.in/telebot.v3"
)

// Handler обрабатывает текстовое сообщение пользователя бота.
type Handler func(userID int64, text string) (string, error)

type WebhookConfig struct {
	PublicURL string
	Listen    string
	Secret    string
}

type Bot struct {
	api     *tele.Bot
	handler Handler
	log     *slog.Logger
}

func New(token string, wh WebhookConfig, handler Handler, log *slog.Logger) (*Bot, error) {
	webhook := &tele.Webhook{
		Listen:      wh.Listen,
		SecretToken: wh.Secret,
		Endpoint: &tele.WebhookEndpoint{
			PublicURL: wh.PublicURL,
		},
	}

	api, err := tele.NewBot(tele.Settings{
		Token:  token,
		Poller: webhook,
	})
	if err != nil {
		return nil, err
	}
	if log == nil {
		log = slog.Default()
	}

	b := &Bot{api: api, handler: handler, log: log}
	b.api.Handle(tele.OnText, b.onText)
	return b, nil
}

func (b *Bot) onText(c tele.Context) error {
	user := c.Sender()
	if user == nil {
		return nil
	}
	text := c.Text()
	if text == "" {
		return nil
	}

	reply, err := b.handler(user.ID, text)
	if err != nil {
		b.log.Error("handler failed", "user_id", user.ID, "err", err)
		return c.Send("Не получилось обработать сообщение. Попробуйте ещё раз.")
	}
	return c.Send(reply)
}

func (b *Bot) Start() {
	b.log.Info("telegram webhook started",
		"username", b.api.Me.Username,
	)
	b.api.Start()
}

func (b *Bot) Stop() {
	b.api.Stop()
}
