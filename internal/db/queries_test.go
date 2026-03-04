package db

import (
	"context"
	"testing"

	"github.com/alexyoung/todoloo/internal/models"
)

func setupTestDB(t *testing.T) *DB {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory database: %v", err)
	}
	return db
}

func TestInsertRawJob(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	jobID, err := db.InsertRawJob(ctx, "buy milk tomorrow")
	if err != nil {
		t.Fatalf("InsertRawJob failed: %v", err)
	}
	if jobID <= 0 {
		t.Errorf("expected positive job ID, got %d", jobID)
	}

	jobs, err := db.GetActionableJobs(ctx, 5)
	if err != nil {
		t.Fatalf("GetActionableJobs failed: %v", err)
	}
	if len(jobs) != 1 {
		t.Errorf("expected 1 job, got %d", len(jobs))
	}
	if jobs[0].RawText != "buy milk tomorrow" {
		t.Errorf("expected 'buy milk tomorrow', got %s", jobs[0].RawText)
	}
	if jobs[0].Status != models.JobStatusPending {
		t.Errorf("expected pending status, got %s", jobs[0].Status)
	}
}

func TestMarkProcessing(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	jobID, _ := db.InsertRawJob(ctx, "test job")
	err := db.MarkProcessing(ctx, jobID)
	if err != nil {
		t.Fatalf("MarkProcessing failed: %v", err)
	}

	jobs, _ := db.GetActionableJobs(ctx, 5)
	if len(jobs) != 0 {
		t.Errorf("expected no pending jobs after marking processing, got %d", len(jobs))
	}
}

func TestMarkDone(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	jobID, _ := db.InsertRawJob(ctx, "test job")
	db.MarkProcessing(ctx, jobID)
	err := db.MarkDone(ctx, jobID)
	if err != nil {
		t.Fatalf("MarkDone failed: %v", err)
	}

	jobs, _ := db.GetQueueStatus(ctx)
	if len(jobs) != 0 {
		t.Errorf("expected no jobs in queue after done, got %d", len(jobs))
	}
}

func TestMarkFailed(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	jobID, _ := db.InsertRawJob(ctx, "test job")
	err := db.MarkFailed(ctx, jobID, "some error")
	if err != nil {
		t.Fatalf("MarkFailed failed: %v", err)
	}

	// Verify job status is failed
	status, err := db.GetQueueStatus(ctx)
	if err != nil {
		t.Fatalf("GetQueueStatus failed: %v", err)
	}
	if len(status) != 1 {
		t.Errorf("expected 1 job in queue, got %d", len(status))
	}
	if status[0].Status != models.JobStatusFailed {
		t.Errorf("expected status 'failed', got %s", status[0].Status)
	}
}

func TestInsertTodo(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	rawID, _ := db.InsertRawJob(ctx, "book a room at the library for 5:30pm")

	parsed := &models.ParsedTodo{
		Title:       "Book library room",
		Description: "For officer meeting",
		Category:    "work",
		DueDate:     "2026-03-04",
		DueTime:     "17:30",
		Urgency:     4,
		Location:    "Library",
		Tags:        []string{"meeting", "booking"},
		Notes:       "Reserve for 2 hours",
	}

	todoID, err := db.InsertTodo(ctx, parsed, rawID)
	if err != nil {
		t.Fatalf("InsertTodo failed: %v", err)
	}
	if todoID <= 0 {
		t.Errorf("expected positive todo ID, got %d", todoID)
	}

	todo, err := db.GetTodoByID(ctx, todoID)
	if err != nil {
		t.Fatalf("GetTodoByID failed: %v", err)
	}
	if todo == nil {
		t.Fatal("expected todo, got nil")
	}
	if todo.Title != "Book library room" {
		t.Errorf("expected 'Book library room', got %s", todo.Title)
	}
	if todo.Category != "work" {
		t.Errorf("expected category 'work', got %s", todo.Category)
	}
	if todo.DueDate != "2026-03-04" {
		t.Errorf("expected due date '2026-03-04', got %s", todo.DueDate)
	}
	if todo.DueTime != "17:30" {
		t.Errorf("expected due time '17:30', got %s", todo.DueTime)
	}
	if todo.Urgency != 4 {
		t.Errorf("expected urgency 4, got %d", todo.Urgency)
	}
	if todo.Location != "Library" {
		t.Errorf("expected location 'Library', got %s", todo.Location)
	}
	if todo.Status != models.TodoStatusOpen {
		t.Errorf("expected status 'open', got %s", todo.Status)
	}
}

func TestQueryTodos(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	parsed1 := &models.ParsedTodo{Title: "Task 1", Category: "work", Urgency: 3, DueDate: "2026-03-04"}
	parsed2 := &models.ParsedTodo{Title: "Task 2", Category: "personal", Urgency: 5, DueDate: "2026-03-10"}
	parsed3 := &models.ParsedTodo{Title: "Task 3", Category: "work", Urgency: 2, DueDate: "2026-03-05"}

	db.InsertTodo(ctx, parsed1, 0)
	db.InsertTodo(ctx, parsed2, 0)
	db.InsertTodo(ctx, parsed3, 0)

	todos, err := db.QueryTodos(ctx, models.TodoFilters{Category: "work"})
	if err != nil {
		t.Fatalf("QueryTodos failed: %v", err)
	}
	if len(todos) != 2 {
		t.Errorf("expected 2 work todos, got %d", len(todos))
	}

	todos, err = db.QueryTodos(ctx, models.TodoFilters{UrgencyMin: 4})
	if err != nil {
		t.Fatalf("QueryTodos failed: %v", err)
	}
	if len(todos) != 1 {
		t.Errorf("expected 1 high urgency todo, got %d", len(todos))
	}

	todos, err = db.QueryTodos(ctx, models.TodoFilters{DueBefore: "2026-03-06"})
	if err != nil {
		t.Fatalf("QueryTodos failed: %v", err)
	}
	if len(todos) != 2 {
		t.Errorf("expected 2 todos due before 2026-03-06, got %d", len(todos))
	}
}

func TestUpdateTodo(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	parsed := &models.ParsedTodo{Title: "Original", Category: "work", Urgency: 3}
	todoID, _ := db.InsertTodo(ctx, parsed, 0)

	err := db.UpdateTodo(ctx, todoID, "Updated", "New description", "personal", "2026-03-15", "10:00", "Office", "", 5, models.TodoStatusOpen)
	if err != nil {
		t.Fatalf("UpdateTodo failed: %v", err)
	}

	todo, _ := db.GetTodoByID(ctx, todoID)
	if todo.Title != "Updated" {
		t.Errorf("expected title 'Updated', got %s", todo.Title)
	}
	if todo.Description != "New description" {
		t.Errorf("expected description 'New description', got %s", todo.Description)
	}
	if todo.Category != "personal" {
		t.Errorf("expected category 'personal', got %s", todo.Category)
	}
	if todo.Urgency != 5 {
		t.Errorf("expected urgency 5, got %d", todo.Urgency)
	}
}

func TestMarkTodoDone(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	parsed := &models.ParsedTodo{Title: "Task to complete"}
	todoID, _ := db.InsertTodo(ctx, parsed, 0)

	err := db.MarkTodoDone(ctx, todoID)
	if err != nil {
		t.Fatalf("MarkTodoDone failed: %v", err)
	}

	todo, _ := db.GetTodoByID(ctx, todoID)
	if todo.Status != models.TodoStatusDone {
		t.Errorf("expected status 'done', got %s", todo.Status)
	}
	if todo.CompletedAt == nil {
		t.Error("expected completed_at to be set")
	}
}

func TestDeleteTodo(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	parsed := &models.ParsedTodo{Title: "Task to delete"}
	todoID, _ := db.InsertTodo(ctx, parsed, 0)

	err := db.DeleteTodo(ctx, todoID)
	if err != nil {
		t.Fatalf("DeleteTodo failed: %v", err)
	}

	todo, _ := db.GetTodoByID(ctx, todoID)
	if todo != nil {
		t.Error("expected nil todo after delete")
	}
}

func TestQueueStatus(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	db.InsertRawJob(ctx, "job 1")
	db.InsertRawJob(ctx, "job 2")

	jobID, _ := db.InsertRawJob(ctx, "job 3")
	db.MarkProcessing(ctx, jobID)
	db.MarkDone(ctx, jobID)

	status, err := db.GetQueueStatus(ctx)
	if err != nil {
		t.Fatalf("GetQueueStatus failed: %v", err)
	}
	if len(status) != 2 {
		t.Errorf("expected 2 jobs in queue, got %d", len(status))
	}
}

func TestRetryJob(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	jobID, _ := db.InsertRawJob(ctx, "failing job")
	db.MarkProcessing(ctx, jobID)
	db.MarkFailed(ctx, jobID, "error")

	err := db.RetryJob(ctx, jobID)
	if err != nil {
		t.Fatalf("RetryJob failed: %v", err)
	}

	jobs, _ := db.GetActionableJobs(ctx, 5)
	if len(jobs) != 1 {
		t.Errorf("expected 1 actionable job after retry, got %d", len(jobs))
	}
}
