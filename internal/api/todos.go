package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/alexyoung/todoloo/internal/db"
	"github.com/alexyoung/todoloo/internal/models"
	"github.com/go-chi/chi/v5"
)

type TodoHandler struct {
	db *db.DB
}

func NewTodoHandler(database *db.DB) *TodoHandler {
	return &TodoHandler{db: database}
}

type RawTodoRequest struct {
	Text string `json:"text"`
}

type RawTodoResponse struct {
	JobID   int64  `json:"job_id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

func (h *TodoHandler) PostRawTodo(w http.ResponseWriter, r *http.Request) {
	var req RawTodoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Text == "" {
		http.Error(w, "text is required", http.StatusBadRequest)
		return
	}

	jobID, err := h.db.InsertRawJob(r.Context(), req.Text)
	if err != nil {
		http.Error(w, "Failed to insert job", http.StatusInternalServerError)
		return
	}

	resp := RawTodoResponse{
		JobID:   jobID,
		Status:  string(models.JobStatusPending),
		Message: "Todo queued for processing",
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(resp)
}

func (h *TodoHandler) GetTodos(w http.ResponseWriter, r *http.Request) {
	filters := models.TodoFilters{
		Category:  r.URL.Query().Get("category"),
		Status:    r.URL.Query().Get("status"),
		DueBefore: r.URL.Query().Get("due_before"),
		DueAfter:  r.URL.Query().Get("due_after"),
		Tags:      r.URL.Query().Get("tags"),
		Query:     r.URL.Query().Get("q"),
		Sort:      r.URL.Query().Get("sort"),
		Order:     r.URL.Query().Get("order"),
		Limit:     50,
		Offset:    0,
	}

	if lim := r.URL.Query().Get("limit"); lim != "" {
		if l, err := strconv.Atoi(lim); err == nil {
			filters.Limit = l
		}
	}
	if off := r.URL.Query().Get("offset"); off != "" {
		if o, err := strconv.Atoi(off); err == nil {
			filters.Offset = o
		}
	}
	if urg := r.URL.Query().Get("urgency_min"); urg != "" {
		if u, err := strconv.Atoi(urg); err == nil {
			filters.UrgencyMin = u
		}
	}

	todos, err := h.db.QueryTodos(r.Context(), filters)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to query todos: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(todos)
}

func (h *TodoHandler) GetTodo(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid todo ID", http.StatusBadRequest)
		return
	}

	todo, err := h.db.GetTodoByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Failed to get todo", http.StatusInternalServerError)
		return
	}
	if todo == nil {
		http.Error(w, "Todo not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(todo)
}

type UpdateTodoRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Category    string `json:"category"`
	DueDate     string `json:"due_date"`
	DueTime     string `json:"due_time"`
	Urgency     int    `json:"urgency"`
	Location    string `json:"location"`
	Recurrence  string `json:"recurrence"`
	Status      string `json:"status"`
}

func (h *TodoHandler) PatchTodo(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid todo ID", http.StatusBadRequest)
		return
	}

	var req UpdateTodoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	existing, err := h.db.GetTodoByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Failed to get todo", http.StatusInternalServerError)
		return
	}
	if existing == nil {
		http.Error(w, "Todo not found", http.StatusNotFound)
		return
	}

	title := req.Title
	if title == "" {
		title = existing.Title
	}
	description := req.Description
	if description == "" {
		description = existing.Description
	}
	category := req.Category
	if category == "" {
		category = existing.Category
	}
	dueDate := req.DueDate
	if dueDate == "" {
		dueDate = existing.DueDate
	}
	dueTime := req.DueTime
	if dueTime == "" {
		dueTime = existing.DueTime
	}
	urgency := req.Urgency
	if urgency == 0 {
		urgency = existing.Urgency
	}
	location := req.Location
	if location == "" {
		location = existing.Location
	}
	recurrence := req.Recurrence
	if recurrence == "" {
		recurrence = existing.Recurrence
	}
	status := models.TodoStatus(req.Status)
	if status == "" {
		status = existing.Status
	}

	err = h.db.UpdateTodo(r.Context(), id, title, description, category, dueDate, dueTime, location, recurrence, urgency, status)
	if err != nil {
		http.Error(w, "Failed to update todo", http.StatusInternalServerError)
		return
	}

	todo, _ := h.db.GetTodoByID(r.Context(), id)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(todo)
}

func (h *TodoHandler) MarkDone(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid todo ID", http.StatusBadRequest)
		return
	}

	err = h.db.MarkTodoDone(r.Context(), id)
	if err != nil {
		http.Error(w, "Failed to mark todo done", http.StatusInternalServerError)
		return
	}

	todo, err := h.db.GetTodoByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Failed to get todo", http.StatusInternalServerError)
		return
	}
	if todo == nil {
		http.Error(w, "Todo not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(todo)
}

func (h *TodoHandler) DeleteTodo(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid todo ID", http.StatusBadRequest)
		return
	}

	err = h.db.DeleteTodo(r.Context(), id)
	if err != nil {
		http.Error(w, "Failed to delete todo", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
