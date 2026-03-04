package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/alexyoung/todoloo/internal/db"
	"github.com/go-chi/chi/v5"
)

type QueueHandler struct {
	db *db.DB
}

func NewQueueHandler(database *db.DB) *QueueHandler {
	return &QueueHandler{db: database}
}

func (h *QueueHandler) GetQueueStatus(w http.ResponseWriter, r *http.Request) {
	jobs, err := h.db.GetQueueStatus(r.Context())
	if err != nil {
		http.Error(w, "Failed to get queue status", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jobs)
}

func (h *QueueHandler) RetryJob(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid job ID", http.StatusBadRequest)
		return
	}

	err = h.db.RetryJob(r.Context(), id)
	if err != nil {
		http.Error(w, "Failed to retry job", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "message": "Job queued for retry"})
}
