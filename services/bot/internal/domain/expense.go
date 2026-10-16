package domain

import (
	"time"

	"github.com/google/uuid"
)

type Expense struct {
	ID        uuid.UUID
	UserID    int64
	Amount    float64
	Category  string
	SpentAt   time.Time
	Comment   *string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CreateExpense struct {
	UserID   int64
	Amount   float64
	Category string
	SpentAt  time.Time
	Comment  *string
}

type UpdateExpense struct {
	Amount   *float64
	Category *string
	SpentAt  *time.Time
	Comment  *string
}

// ListFilter — фильтры списка трат.
// UserID == nil → все пользователи (веб); иначе только этот user (бот).
// Page — 1-based; PageSize — размер страницы.
type ListFilter struct {
	UserID   *int64
	From     *time.Time
	To       *time.Time
	Category *string
	Page     int
	PageSize int
}
