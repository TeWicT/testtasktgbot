package postgres

import (
	"context"
	"errors"
	"fmt"

	"expenses/bot/internal/domain"
	"expenses/bot/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ExpensesRepo struct {
	pool *pgxpool.Pool
}

func NewExpensesRepo(pool *pgxpool.Pool) repository.ExpenseRepo {
	return &ExpensesRepo{pool: pool}
}

func (r *ExpensesRepo) Create(ctx context.Context, in domain.CreateExpense) (*domain.Expense, error) {
	const q = `
		INSERT INTO expenses (user_id, amount, category, spent_at, comment)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, amount, category, spent_at, comment, created_at, updated_at
	`
	return scanExpense(r.pool.QueryRow(ctx, q,
		in.UserID,
		in.Amount,
		in.Category,
		in.SpentAt,
		in.Comment,
	))
}

func (r *ExpensesRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Expense, error) {
	const q = `
		SELECT id, user_id, amount, category, spent_at, comment, created_at, updated_at
		FROM expenses
		WHERE id = $1
	`
	exp, err := scanExpense(r.pool.QueryRow(ctx, q, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, repository.ErrNotFound
	}
	return exp, err
}

func buildWhere(f domain.ListFilter) (where string, args []any) {
	where = "WHERE 1=1"
	args = make([]any, 0, 4)
	n := 1

	if f.UserID != nil {
		where += fmt.Sprintf(" AND user_id = $%d", n)
		args = append(args, *f.UserID)
		n++
	}
	if f.From != nil {
		where += fmt.Sprintf(" AND spent_at >= $%d", n)
		args = append(args, *f.From)
		n++
	}
	if f.To != nil {
		where += fmt.Sprintf(" AND spent_at <= $%d", n)
		args = append(args, *f.To)
		n++
	}
	if f.Category != nil {
		where += fmt.Sprintf(" AND category = $%d", n)
		args = append(args, *f.Category)
	}
	return where, args
}

func normalizePage(f domain.ListFilter) (page, pageSize int) {
	page = f.Page
	if page < 1 {
		page = 1
	}
	pageSize = f.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func (r *ExpensesRepo) List(ctx context.Context, f domain.ListFilter) ([]domain.Expense, error) {
	page, pageSize := normalizePage(f)
	offset := (page - 1) * pageSize

	where, args := buildWhere(f)
	n := len(args) + 1
	q := `
		SELECT id, user_id, amount, category, spent_at, comment, created_at, updated_at
		FROM expenses
		` + where + fmt.Sprintf(`
		ORDER BY spent_at DESC, created_at DESC
		LIMIT $%d OFFSET $%d
	`, n, n+1)
	args = append(args, pageSize, offset)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.Expense, 0)
	for rows.Next() {
		exp, err := scanExpense(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *exp)
	}
	return out, rows.Err()
}

func (r *ExpensesRepo) Count(ctx context.Context, f domain.ListFilter) (int64, error) {
	where, args := buildWhere(f)
	q := `SELECT COUNT(*) FROM expenses ` + where
	var total int64
	if err := r.pool.QueryRow(ctx, q, args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

func (r *ExpensesRepo) ListUserIDs(ctx context.Context) ([]int64, error) {
	const q = `SELECT DISTINCT user_id FROM expenses ORDER BY user_id`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (r *ExpensesRepo) UpdateForUser(ctx context.Context, userID int64, id uuid.UUID, in domain.UpdateExpense) (*domain.Expense, error) {
	const q = `
		UPDATE expenses
		SET
			amount    = COALESCE($1, amount),
			category  = COALESCE($2, category),
			spent_at  = COALESCE($3, spent_at),
			comment   = COALESCE($4, comment),
			updated_at = now()
		WHERE id = $5 AND user_id = $6
		RETURNING id, user_id, amount, category, spent_at, comment, created_at, updated_at
	`
	exp, err := scanExpense(r.pool.QueryRow(ctx, q,
		in.Amount, in.Category, in.SpentAt, in.Comment, id, userID,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, repository.ErrNotFound
	}
	return exp, err
}

func (r *ExpensesRepo) Update(ctx context.Context, id uuid.UUID, in domain.UpdateExpense) (*domain.Expense, error) {
	const q = `
		UPDATE expenses
		SET
			amount    = COALESCE($1, amount),
			category  = COALESCE($2, category),
			spent_at  = COALESCE($3, spent_at),
			comment   = COALESCE($4, comment),
			updated_at = now()
		WHERE id = $5
		RETURNING id, user_id, amount, category, spent_at, comment, created_at, updated_at
	`
	exp, err := scanExpense(r.pool.QueryRow(ctx, q,
		in.Amount, in.Category, in.SpentAt, in.Comment, id,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, repository.ErrNotFound
	}
	return exp, err
}

func (r *ExpensesRepo) DeleteForUser(ctx context.Context, userID int64, id uuid.UUID) error {
	const q = `DELETE FROM expenses WHERE id = $1 AND user_id = $2`
	tag, err := r.pool.Exec(ctx, q, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return repository.ErrNotFound
	}
	return nil
}

func (r *ExpensesRepo) Delete(ctx context.Context, id uuid.UUID) error {
	const q = `DELETE FROM expenses WHERE id = $1`
	tag, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return repository.ErrNotFound
	}
	return nil
}

func (r *ExpensesRepo) Latest(ctx context.Context, userID int64) (*domain.Expense, error) {
	const q = `
		SELECT id, user_id, amount, category, spent_at, comment, created_at, updated_at
		FROM expenses
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`
	exp, err := scanExpense(r.pool.QueryRow(ctx, q, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, repository.ErrNotFound
	}
	return exp, err
}

func (r *ExpensesRepo) Sum(ctx context.Context, f domain.ListFilter) (float64, error) {
	where, args := buildWhere(f)
	q := `SELECT COALESCE(SUM(amount), 0) FROM expenses ` + where
	var sum float64
	if err := r.pool.QueryRow(ctx, q, args...).Scan(&sum); err != nil {
		return 0, err
	}
	return sum, nil
}

type scannable interface {
	Scan(dest ...any) error
}

func scanExpense(row scannable) (*domain.Expense, error) {
	var e domain.Expense
	err := row.Scan(
		&e.ID,
		&e.UserID,
		&e.Amount,
		&e.Category,
		&e.SpentAt,
		&e.Comment,
		&e.CreatedAt,
		&e.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &e, nil
}
