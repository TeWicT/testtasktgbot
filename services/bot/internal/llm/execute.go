package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"expenses/bot/internal/domain"
	"expenses/bot/internal/repository"

	"github.com/google/uuid"
	openai "github.com/sashabaranov/go-openai"
)

func (c *Client) executeTool(ctx context.Context, userID int64, call openai.ToolCall, now time.Time) (string, error) {
	switch call.Function.Name {
	case "add_expense":
		return c.execAddExpense(ctx, userID, call.Function.Arguments, now)
	case "list_expenses":
		return c.execListExpenses(ctx, userID, call.Function.Arguments, now)
	case "sum_expenses":
		return c.execSumExpenses(ctx, userID, call.Function.Arguments, now)
	case "update_latest_expense":
		return c.execUpdateLatest(ctx, userID, call.Function.Arguments, now)
	case "update_expense":
		return c.execUpdateExpense(ctx, userID, call.Function.Arguments, now)
	case "delete_expense":
		return c.execDeleteExpense(ctx, userID, call.Function.Arguments)
	default:
		return "", fmt.Errorf("unknown tool %s", call.Function.Name)
	}
}

func (c *Client) execAddExpense(ctx context.Context, userID int64, raw string, now time.Time) (string, error) {
	var args addExpenseArgs
	if err := json.Unmarshal([]byte(raw), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if args.Amount <= 0 {
		return "", fmt.Errorf("amount must be positive")
	}
	if _, ok := allowedCategories[args.Category]; !ok {
		return "", fmt.Errorf("unknown category %q", args.Category)
	}

	spentAt, err := parseDate(args.Date, now, c.loc)
	if err != nil {
		return "", err
	}

	var comment *string
	if s := strings.TrimSpace(args.Comment); s != "" {
		comment = &s
	}

	exp, err := c.repo.Create(ctx, domain.CreateExpense{
		UserID:   userID,
		Amount:   args.Amount,
		Category: args.Category,
		SpentAt:  spentAt,
		Comment:  comment,
	})
	if err != nil {
		return "", err
	}

	return toJSON(expenseView(exp)), nil
}

func (c *Client) execListExpenses(ctx context.Context, userID int64, raw string, now time.Time) (string, error) {
	var args listArgs
	if err := json.Unmarshal([]byte(raw), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	f, err := c.listFilter(userID, args, now)
	if err != nil {
		return "", err
	}
	items, err := c.repo.List(ctx, f)
	if err != nil {
		return "", err
	}

	views := make([]map[string]any, 0, len(items))
	for i := range items {
		views = append(views, expenseView(&items[i]))
	}
	return toJSON(map[string]any{"items": views, "count": len(views)}), nil
}

func (c *Client) execSumExpenses(ctx context.Context, userID int64, raw string, now time.Time) (string, error) {
	var args listArgs
	if err := json.Unmarshal([]byte(raw), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	f, err := c.listFilter(userID, args, now)
	if err != nil {
		return "", err
	}
	sum, err := c.repo.Sum(ctx, f)
	if err != nil {
		return "", err
	}
	return toJSON(map[string]any{"sum": sum, "currency": "RUB"}), nil
}

func (c *Client) execUpdateLatest(ctx context.Context, userID int64, raw string, now time.Time) (string, error) {
	var args updateLatestArgs
	if err := json.Unmarshal([]byte(raw), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	latest, err := c.repo.Latest(ctx, userID)
	if err != nil {
		return "", err
	}

	in, err := c.buildUpdate(args.Amount, args.Category, args.Date, args.Comment, now)
	if err != nil {
		return "", err
	}

	exp, err := c.repo.UpdateForUser(ctx, userID, latest.ID, in)
	if err != nil {
		return "", err
	}
	return toJSON(expenseView(exp)), nil
}

func (c *Client) execUpdateExpense(ctx context.Context, userID int64, raw string, now time.Time) (string, error) {
	var args updateArgs
	if err := json.Unmarshal([]byte(raw), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	id, err := uuid.Parse(args.ID)
	if err != nil {
		return "", fmt.Errorf("invalid id")
	}

	in, err := c.buildUpdate(args.Amount, args.Category, args.Date, args.Comment, now)
	if err != nil {
		return "", err
	}

	exp, err := c.repo.UpdateForUser(ctx, userID, id, in)
	if err != nil {
		return "", err
	}
	return toJSON(expenseView(exp)), nil
}

func (c *Client) execDeleteExpense(ctx context.Context, userID int64, raw string) (string, error) {
	var args deleteArgs
	if err := json.Unmarshal([]byte(raw), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if args.ID != "" {
		id, err := uuid.Parse(args.ID)
		if err != nil {
			return "", fmt.Errorf("invalid id")
		}
		if err := c.repo.DeleteForUser(ctx, userID, id); err != nil {
			return "", err
		}
		return toJSON(map[string]any{"deleted_id": args.ID}), nil
	}

	hint := strings.TrimSpace(args.CommentHint)
	var category *string
	if args.Category != "" {
		if _, ok := allowedCategories[args.Category]; !ok {
			return "", fmt.Errorf("unknown category %q", args.Category)
		}
		cat := args.Category
		category = &cat
	}
	if hint == "" && category == nil {
		return "", fmt.Errorf("укажите id, comment_hint или category")
	}

	uid := userID
	items, err := c.repo.List(ctx, domain.ListFilter{
		UserID:   &uid,
		Category: category,
		Page:     1,
		PageSize: 50,
	})
	if err != nil {
		return "", err
	}

	hintLower := strings.ToLower(hint)
	for i := range items {
		exp := &items[i]
		if hint != "" {
			if exp.Comment == nil || !strings.Contains(strings.ToLower(*exp.Comment), hintLower) {
				continue
			}
		}
		if err := c.repo.DeleteForUser(ctx, userID, exp.ID); err != nil {
			return "", err
		}
		return toJSON(map[string]any{"deleted": expenseView(exp)}), nil
	}

	return "", repository.ErrNotFound
}

func (c *Client) buildUpdate(amount *float64, category, date string, comment *string, now time.Time) (domain.UpdateExpense, error) {
	in := domain.UpdateExpense{Amount: amount, Comment: comment}
	if amount != nil && *amount <= 0 {
		return in, fmt.Errorf("amount must be positive")
	}
	if category != "" {
		if _, ok := allowedCategories[category]; !ok {
			return in, fmt.Errorf("unknown category %q", category)
		}
		cat := category
		in.Category = &cat
	}
	if date != "" {
		d, err := parseDate(date, now, c.loc)
		if err != nil {
			return in, err
		}
		in.SpentAt = &d
	}
	return in, nil
}

func (c *Client) listFilter(userID int64, args listArgs, now time.Time) (domain.ListFilter, error) {
	uid := userID
	pageSize := args.Limit
	if pageSize <= 0 {
		pageSize = 50
	}
	f := domain.ListFilter{UserID: &uid, Page: 1, PageSize: pageSize}
	if args.Category != "" {
		if _, ok := allowedCategories[args.Category]; !ok {
			return f, fmt.Errorf("unknown category %q", args.Category)
		}
		cat := args.Category
		f.Category = &cat
	}

	from, to, err := periodRange(args.Period, args.From, args.To, now, c.loc)
	if err != nil {
		return f, err
	}
	f.From, f.To = from, to
	return f, nil
}

func expenseView(e *domain.Expense) map[string]any {
	comment := ""
	if e.Comment != nil {
		comment = *e.Comment
	}
	return map[string]any{
		"id":       e.ID.String(),
		"amount":   e.Amount,
		"category": e.Category,
		"spent_at": e.SpentAt.Format("2006-01-02"),
		"comment":  comment,
	}
}

func toJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
