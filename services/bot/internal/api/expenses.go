package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"expenses/bot/internal/domain"
	"expenses/bot/internal/repository"

	"github.com/google/uuid"
)

type badRequestError struct{ msg string }

func (e *badRequestError) Error() string { return e.msg }

func errBadRequest(msg string) error { return &badRequestError{msg: msg} }

var allowedCategories = map[string]struct{}{
	"еда": {}, "транспорт": {}, "жильё": {}, "развлечения": {},
	"здоровье": {}, "покупки": {}, "другое": {},
}

type ExpensesHandler struct {
	repo repository.ExpenseRepo
	loc  *time.Location
}

func NewExpensesHandler(repo repository.ExpenseRepo, loc *time.Location) *ExpensesHandler {
	if loc == nil {
		loc = time.UTC
	}
	return &ExpensesHandler{repo: repo, loc: loc}
}

type expenseDTO struct {
	ID        string  `json:"id"`
	UserID    int64   `json:"user_id"`
	Amount    float64 `json:"amount"`
	Category  string  `json:"category"`
	SpentAt   string  `json:"spent_at"`
	Comment   *string `json:"comment,omitempty"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

type createExpenseRequest struct {
	UserID   int64   `json:"user_id"`
	Amount   float64 `json:"amount"`
	Category string  `json:"category"`
	SpentAt  string  `json:"spent_at"`
	Comment  *string `json:"comment"`
}

type updateExpenseRequest struct {
	Amount   *float64 `json:"amount"`
	Category *string  `json:"category"`
	SpentAt  *string  `json:"spent_at"`
	Comment  *string  `json:"comment"`
}

func toDTO(e *domain.Expense) expenseDTO {
	return expenseDTO{
		ID:        e.ID.String(),
		UserID:    e.UserID,
		Amount:    e.Amount,
		Category:  e.Category,
		SpentAt:   e.SpentAt.Format("2006-01-02"),
		Comment:   e.Comment,
		CreatedAt: e.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: e.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func (h *ExpensesHandler) parseListFilter(r *http.Request) (domain.ListFilter, error) {
	f := domain.ListFilter{Page: 1, PageSize: 20}

	if v := r.URL.Query().Get("user_id"); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return f, errBadRequest("invalid user_id")
		}
		f.UserID = &id
	}
	if v := r.URL.Query().Get("category"); v != "" {
		if _, ok := allowedCategories[v]; !ok {
			return f, errBadRequest("invalid category")
		}
		cat := v
		f.Category = &cat
	}
	if v := r.URL.Query().Get("from"); v != "" {
		t, err := time.ParseInLocation("2006-01-02", v, h.loc)
		if err != nil {
			return f, errBadRequest("invalid from")
		}
		f.From = &t
	}
	if v := r.URL.Query().Get("to"); v != "" {
		t, err := time.ParseInLocation("2006-01-02", v, h.loc)
		if err != nil {
			return f, errBadRequest("invalid to")
		}
		f.To = &t
	}
	if v := r.URL.Query().Get("page"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			return f, errBadRequest("invalid page")
		}
		f.Page = n
	}
	if v := r.URL.Query().Get("page_size"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			return f, errBadRequest("invalid page_size")
		}
		f.PageSize = n
	}
	return f, nil
}

func writeFilterError(w http.ResponseWriter, err error) {
	var br *badRequestError
	if errors.As(err, &br) {
		writeError(w, http.StatusBadRequest, br.msg)
		return
	}
	writeError(w, http.StatusBadRequest, "invalid filters")
}

func (h *ExpensesHandler) List(w http.ResponseWriter, r *http.Request) {
	f, err := h.parseListFilter(r)
	if err != nil {
		writeFilterError(w, err)
		return
	}

	items, err := h.repo.List(r.Context(), f)
	if err != nil {
		writeRepoError(w, err)
		return
	}
	total, err := h.repo.Count(r.Context(), f)
	if err != nil {
		writeRepoError(w, err)
		return
	}

	pageSize := f.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	page := f.Page
	if page < 1 {
		page = 1
	}
	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))
	if total == 0 {
		totalPages = 0
	}

	out := make([]expenseDTO, 0, len(items))
	for i := range items {
		out = append(out, toDTO(&items[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items":       out,
		"page":        page,
		"page_size":   pageSize,
		"total":       total,
		"total_pages": totalPages,
	})
}

func (h *ExpensesHandler) Sum(w http.ResponseWriter, r *http.Request) {
	f, err := h.parseListFilter(r)
	if err != nil {
		writeFilterError(w, err)
		return
	}

	sum, err := h.repo.Sum(r.Context(), f)
	if err != nil {
		writeRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sum": sum, "currency": "RUB"})
}

func (h *ExpensesHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	ids, err := h.repo.ListUserIDs(r.Context())
	if err != nil {
		writeRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user_ids": ids})
}

func (h *ExpensesHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	exp, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		writeRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toDTO(exp))
}

func (h *ExpensesHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createExpenseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.UserID == 0 {
		writeError(w, http.StatusBadRequest, "user_id is required")
		return
	}
	if req.Amount <= 0 {
		writeError(w, http.StatusBadRequest, "amount must be positive")
		return
	}
	if _, ok := allowedCategories[req.Category]; !ok {
		writeError(w, http.StatusBadRequest, "invalid category")
		return
	}
	spentAt, err := time.ParseInLocation("2006-01-02", req.SpentAt, h.loc)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid spent_at, use YYYY-MM-DD")
		return
	}

	exp, err := h.repo.Create(r.Context(), domain.CreateExpense{
		UserID:   req.UserID,
		Amount:   req.Amount,
		Category: req.Category,
		SpentAt:  spentAt,
		Comment:  req.Comment,
	})
	if err != nil {
		writeRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toDTO(exp))
}

func (h *ExpensesHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var req updateExpenseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	in := domain.UpdateExpense{Amount: req.Amount, Category: req.Category, Comment: req.Comment}
	if req.Amount != nil && *req.Amount <= 0 {
		writeError(w, http.StatusBadRequest, "amount must be positive")
		return
	}
	if req.Category != nil {
		if _, ok := allowedCategories[*req.Category]; !ok {
			writeError(w, http.StatusBadRequest, "invalid category")
			return
		}
	}
	if req.SpentAt != nil {
		t, err := time.ParseInLocation("2006-01-02", *req.SpentAt, h.loc)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid spent_at, use YYYY-MM-DD")
			return
		}
		in.SpentAt = &t
	}

	exp, err := h.repo.Update(r.Context(), id, in)
	if err != nil {
		writeRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toDTO(exp))
}

func (h *ExpensesHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.repo.Delete(r.Context(), id); err != nil {
		writeRepoError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
