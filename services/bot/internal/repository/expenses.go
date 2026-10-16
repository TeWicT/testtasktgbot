package repository

import (
	"context"
	"errors"

	"expenses/bot/internal/domain"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("expense not found")

type ExpenseRepo interface {
	Create(ctx context.Context, in domain.CreateExpense) (*domain.Expense, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Expense, error)
	List(ctx context.Context, f domain.ListFilter) ([]domain.Expense, error)
	Count(ctx context.Context, f domain.ListFilter) (int64, error)
	ListUserIDs(ctx context.Context) ([]int64, error)
	// UpdateForUser / DeleteForUser — только свои траты (бот).
	UpdateForUser(ctx context.Context, userID int64, id uuid.UUID, in domain.UpdateExpense) (*domain.Expense, error)
	DeleteForUser(ctx context.Context, userID int64, id uuid.UUID) error
	// Update / Delete — любая трата (веб).
	Update(ctx context.Context, id uuid.UUID, in domain.UpdateExpense) (*domain.Expense, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Latest(ctx context.Context, userID int64) (*domain.Expense, error)
	Sum(ctx context.Context, f domain.ListFilter) (float64, error)
}
