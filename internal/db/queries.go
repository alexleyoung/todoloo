package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/alexyoung/todoloo/internal/models"
)

func (d *DB) InsertRawJob(ctx context.Context, text string) (int64, error) {
	result, err := d.ExecContext(ctx,
		"INSERT INTO raw_queue (raw_text, status) VALUES (?, ?)",
		text, models.JobStatusPending,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to insert raw job: %w", err)
	}
	return result.LastInsertId()
}

func (d *DB) GetActionableJobs(ctx context.Context, maxRetries int) ([]models.RawJob, error) {
	query := `
		SELECT id, raw_text, submitted_at, status, attempts, last_attempted, error_msg
		FROM raw_queue
		WHERE status = 'pending'
		   OR (status = 'failed' AND attempts < ? AND last_attempted < datetime('now', '-' || (30 * pow(2, attempts)) || ' seconds'))
		ORDER BY submitted_at ASC
		LIMIT 10
	`
	rows, err := d.QueryContext(ctx, query, maxRetries)
	if err != nil {
		return nil, fmt.Errorf("failed to get actionable jobs: %w", err)
	}
	defer rows.Close()

	var jobs []models.RawJob
	for rows.Next() {
		var job models.RawJob
		var lastAttempted sql.NullTime
		var errorMsg sql.NullString
		err := rows.Scan(&job.ID, &job.RawText, &job.SubmittedAt, &job.Status, &job.Attempts, &lastAttempted, &errorMsg)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job: %w", err)
		}
		if lastAttempted.Valid {
			job.LastAttempted = &lastAttempted.Time
		}
		if errorMsg.Valid {
			job.ErrorMsg = &errorMsg.String
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}

func (d *DB) MarkProcessing(ctx context.Context, id int64) error {
	_, err := d.ExecContext(ctx,
		"UPDATE raw_queue SET status = ?, attempts = attempts + 1, last_attempted = ? WHERE id = ?",
		models.JobStatusProcessing, time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("failed to mark processing: %w", err)
	}
	return nil
}

func (d *DB) MarkDone(ctx context.Context, id int64) error {
	_, err := d.ExecContext(ctx,
		"UPDATE raw_queue SET status = ?, last_attempted = ? WHERE id = ?",
		models.JobStatusDone, time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("failed to mark done: %w", err)
	}
	return nil
}

func (d *DB) MarkFailed(ctx context.Context, id int64, errMsg string) error {
	_, err := d.ExecContext(ctx,
		"UPDATE raw_queue SET status = ?, error_msg = ?, last_attempted = ? WHERE id = ?",
		models.JobStatusFailed, errMsg, time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("failed to mark failed: %w", err)
	}
	return nil
}

func (d *DB) InsertTodo(ctx context.Context, parsed *models.ParsedTodo, rawID int64) (int64, error) {
	tagsJSON, _ := json.Marshal(parsed.Tags)
	now := time.Now()

	result, err := d.ExecContext(ctx, `
		INSERT INTO todos (
			raw_id, title, description, category, due_date, due_time,
			urgency, tags, location, recurrence, status, source_text,
			created_at, processed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		rawID,
		parsed.Title,
		parsed.Description,
		parsed.Category,
		parsed.DueDate,
		parsed.DueTime,
		parsed.Urgency,
		string(tagsJSON),
		parsed.Location,
		parsed.Recurrence,
		models.TodoStatusOpen,
		parsed.Notes,
		now,
		now,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to insert todo: %w", err)
	}
	return result.LastInsertId()
}

func (d *DB) GetTodoByID(ctx context.Context, id int64) (*models.Todo, error) {
	var todo models.Todo
	var processedAt sql.NullTime
	var completedAt sql.NullTime

	err := d.QueryRowContext(ctx, `
		SELECT id, raw_id, title, description, category, due_date, due_time,
			   urgency, tags, location, recurrence, status, source_text,
			   created_at, processed_at, completed_at
		FROM todos WHERE id = ?
	`, id).Scan(
		&todo.ID, &todo.RawID, &todo.Title, &todo.Description, &todo.Category,
		&todo.DueDate, &todo.DueTime, &todo.Urgency, &todo.Tags, &todo.Location,
		&todo.Recurrence, &todo.Status, &todo.SourceText, &todo.CreatedAt,
		&processedAt, &completedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get todo: %w", err)
	}
	if processedAt.Valid {
		todo.ProcessedAt = &processedAt.Time
	}
	if completedAt.Valid {
		todo.CompletedAt = &completedAt.Time
	}
	return &todo, nil
}

func (d *DB) QueryTodos(ctx context.Context, filters models.TodoFilters) ([]models.Todo, error) {
	query := "SELECT id, raw_id, title, description, category, due_date, due_time, urgency, tags, location, recurrence, status, source_text, created_at, processed_at, completed_at FROM todos WHERE 1=1"
	args := []interface{}{}

	if filters.Category != "" {
		query += " AND category = ?"
		args = append(args, filters.Category)
	}
	if filters.Status != "" {
		query += " AND status = ?"
		args = append(args, filters.Status)
	}
	if filters.DueBefore != "" {
		query += " AND due_date <= ?"
		args = append(args, filters.DueBefore)
	}
	if filters.DueAfter != "" {
		query += " AND due_date >= ?"
		args = append(args, filters.DueAfter)
	}
	if filters.UrgencyMin > 0 {
		query += " AND urgency >= ?"
		args = append(args, filters.UrgencyMin)
	}
	if filters.Tags != "" {
		tags := strings.Split(filters.Tags, ",")
		for _, tag := range tags {
			query += " AND tags LIKE ?"
			args = append(args, "%"+strings.TrimSpace(tag)+"%")
		}
	}
	if filters.Query != "" {
		query += " AND (title LIKE ? OR description LIKE ? OR notes LIKE ?)"
		searchTerm := "%" + filters.Query + "%"
		args = append(args, searchTerm, searchTerm, searchTerm)
	}

	sortCol := "created_at"
	switch filters.Sort {
	case "due_date":
		sortCol = "due_date"
	case "urgency":
		sortCol = "urgency"
	case "created_at":
		sortCol = "created_at"
	}
	order := "DESC"
	if filters.Order == "asc" {
		order = "ASC"
	}
	query += fmt.Sprintf(" ORDER BY %s %s", sortCol, order)

	if filters.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filters.Limit)
	} else {
		query += " LIMIT 50"
	}
	if filters.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, filters.Offset)
	}

	rows, err := d.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query todos: %w", err)
	}
	defer rows.Close()

	todos := []models.Todo{}
	for rows.Next() {
		var todo models.Todo
		var processedAt sql.NullTime
		var completedAt sql.NullTime
		err := rows.Scan(
			&todo.ID, &todo.RawID, &todo.Title, &todo.Description, &todo.Category,
			&todo.DueDate, &todo.DueTime, &todo.Urgency, &todo.Tags, &todo.Location,
			&todo.Recurrence, &todo.Status, &todo.SourceText, &todo.CreatedAt,
			&processedAt, &completedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan todo: %w", err)
		}
		if processedAt.Valid {
			todo.ProcessedAt = &processedAt.Time
		}
		if completedAt.Valid {
			todo.CompletedAt = &completedAt.Time
		}
		todos = append(todos, todo)
	}
	return todos, nil
}

func (d *DB) UpdateTodo(ctx context.Context, id int64, title, description, category, dueDate, dueTime, location, recurrence string, urgency int, status models.TodoStatus) error {
	_, err := d.ExecContext(ctx, `
		UPDATE todos SET
			title = ?, description = ?, category = ?, due_date = ?, due_time = ?,
			urgency = ?, location = ?, recurrence = ?, status = ?
		WHERE id = ?
	`, title, description, category, dueDate, dueTime, urgency, location, recurrence, status, id)
	if err != nil {
		return fmt.Errorf("failed to update todo: %w", err)
	}
	return nil
}

func (d *DB) MarkTodoDone(ctx context.Context, id int64) error {
	_, err := d.ExecContext(ctx,
		"UPDATE todos SET status = ?, completed_at = ? WHERE id = ?",
		models.TodoStatusDone, time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("failed to mark todo done: %w", err)
	}
	return nil
}

func (d *DB) DeleteTodo(ctx context.Context, id int64) error {
	_, err := d.ExecContext(ctx, "DELETE FROM todos WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete todo: %w", err)
	}
	return nil
}

func (d *DB) GetQueueStatus(ctx context.Context) ([]models.RawJob, error) {
	rows, err := d.QueryContext(ctx, `
		SELECT id, raw_text, submitted_at, status, attempts, last_attempted, error_msg
		FROM raw_queue
		WHERE status != 'done'
		ORDER BY submitted_at DESC
		LIMIT 50
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get queue status: %w", err)
	}
	defer rows.Close()

	var jobs []models.RawJob
	for rows.Next() {
		var job models.RawJob
		var lastAttempted sql.NullTime
		var errorMsg sql.NullString
		err := rows.Scan(&job.ID, &job.RawText, &job.SubmittedAt, &job.Status, &job.Attempts, &lastAttempted, &errorMsg)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job: %w", err)
		}
		if lastAttempted.Valid {
			job.LastAttempted = &lastAttempted.Time
		}
		if errorMsg.Valid {
			job.ErrorMsg = &errorMsg.String
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}

func (d *DB) RetryJob(ctx context.Context, id int64) error {
	_, err := d.ExecContext(ctx,
		"UPDATE raw_queue SET status = ?, attempts = 0, error_msg = NULL WHERE id = ?",
		models.JobStatusPending, id,
	)
	if err != nil {
		return fmt.Errorf("failed to retry job: %w", err)
	}
	return nil
}
