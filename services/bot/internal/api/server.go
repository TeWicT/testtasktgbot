package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"expenses/bot/internal/repository"
)

type Server struct {
	httpServer *http.Server
	log        *slog.Logger
}

func NewServer(addr string, repo repository.ExpenseRepo, log *slog.Logger) *Server {
	if log == nil {
		log = slog.Default()
	}

	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		loc = time.UTC
	}

	expenses := NewExpensesHandler(repo, loc)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", Health())
	mux.HandleFunc("GET /api/users", expenses.ListUsers)
	mux.HandleFunc("GET /api/expenses", expenses.List)
	mux.HandleFunc("GET /api/expenses/sum", expenses.Sum)
	mux.HandleFunc("POST /api/expenses", expenses.Create)
	mux.HandleFunc("GET /api/expenses/{id}", expenses.Get)
	mux.HandleFunc("PUT /api/expenses/{id}", expenses.Update)
	mux.HandleFunc("PATCH /api/expenses/{id}", expenses.Update)
	mux.HandleFunc("DELETE /api/expenses/{id}", expenses.Delete)

	return &Server{
		httpServer: &http.Server{
			Addr:              addr,
			Handler:           withCORS(mux),
			ReadHeaderTimeout: 10 * time.Second,
		},
		log: log,
	}
}

func (s *Server) Start() error {
	s.log.Info("api server listening", "addr", s.httpServer.Addr)
	err := s.httpServer.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
