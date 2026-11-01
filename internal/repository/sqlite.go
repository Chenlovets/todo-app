// Package repository persists tasks in SQLite.
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	_ "modernc.org/sqlite" // registers the "sqlite" driver

	"todoapp/internal/service"
)

const schema = `CREATE TABLE IF NOT EXISTS tasks (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	title      TEXT    NOT NULL,
	completed  INTEGER NOT NULL DEFAULT 0,
	created_at TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP
)`

// SQLite is a task store backed by a SQLite database file.
type SQLite struct {
	db *sql.DB
}

// Open opens (and if needed creates) the database at path.
func Open(path string) (*SQLite, error) {
	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	// SQLite allows one writer at a time; one connection avoids "database is locked".
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("init schema: %w", err)
	}
	return &SQLite{db: db}, nil
}

// Close closes the database.
func (s *SQLite) Close() error { return s.db.Close() }

// List returns all tasks, oldest first.
func (s *SQLite) List(ctx context.Context) ([]service.Task, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, title, completed FROM tasks ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()
	tasks := []service.Task{}
	for rows.Next() {
		var t service.Task
		if err := rows.Scan(&t.ID, &t.Title, &t.Completed); err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	return tasks, nil
}

// Create inserts a new, incomplete task.
func (s *SQLite) Create(ctx context.Context, title string) (service.Task, error) {
	res, err := s.db.ExecContext(ctx, `INSERT INTO tasks (title) VALUES (?)`, title)
	if err != nil {
		return service.Task{}, fmt.Errorf("create task: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return service.Task{}, fmt.Errorf("create task: %w", err)
	}
	return service.Task{ID: id, Title: title}, nil
}

// Get returns one task or service.ErrNotFound.
func (s *SQLite) Get(ctx context.Context, id int64) (service.Task, error) {
	var t service.Task
	err := s.db.QueryRowContext(ctx, `SELECT id, title, completed FROM tasks WHERE id = ?`, id).
		Scan(&t.ID, &t.Title, &t.Completed)
	if errors.Is(err, sql.ErrNoRows) {
		return service.Task{}, service.ErrNotFound
	}
	if err != nil {
		return service.Task{}, fmt.Errorf("get task %d: %w", id, err)
	}
	return t, nil
}

// UpdateTitle sets a task's title.
func (s *SQLite) UpdateTitle(ctx context.Context, id int64, title string) error {
	return s.exec(ctx, id, `UPDATE tasks SET title = ? WHERE id = ?`, title, id)
}

// SetCompleted sets a task's completed flag.
func (s *SQLite) SetCompleted(ctx context.Context, id int64, completed bool) error {
	return s.exec(ctx, id, `UPDATE tasks SET completed = ? WHERE id = ?`, completed, id)
}

// Delete removes a task.
func (s *SQLite) Delete(ctx context.Context, id int64) error {
	return s.exec(ctx, id, `DELETE FROM tasks WHERE id = ?`, id)
}

// exec runs a statement and maps "no rows affected" to service.ErrNotFound.
func (s *SQLite) exec(ctx context.Context, id int64, query string, args ...any) error {
	res, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("modify task %d: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("modify task %d: %w", id, err)
	}
	if n == 0 {
		return service.ErrNotFound
	}
	return nil
}
