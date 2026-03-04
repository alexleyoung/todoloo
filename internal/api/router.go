package api

import (
	"net/http"

	"github.com/alexyoung/todoloo/internal/db"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type ServerConfig struct {
	Host string
	Port int
}

func NewServerConfig() ServerConfig {
	return ServerConfig{
		Host: "127.0.0.1",
		Port: 8080,
	}
}

func Router(database *db.DB) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(RequestLogger)
	r.Use(PanicRecovery)

	todoHandler := NewTodoHandler(database)
	queueHandler := NewQueueHandler(database)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	r.Route("/todos", func(r chi.Router) {
		r.Post("/raw", todoHandler.PostRawTodo)
		r.Get("/", todoHandler.GetTodos)
		r.Get("/{id}", todoHandler.GetTodo)
		r.Patch("/{id}", todoHandler.PatchTodo)
		r.Delete("/{id}", todoHandler.DeleteTodo)
		r.Post("/{id}/done", todoHandler.MarkDone)
	})

	r.Route("/queue", func(r chi.Router) {
		r.Get("/status", queueHandler.GetQueueStatus)
		r.Post("/{id}/retry", queueHandler.RetryJob)
	})

	return r
}

func Serve(addr string, handler http.Handler) error {
	return http.ListenAndServe(addr, handler)
}
