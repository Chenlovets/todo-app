// Package service holds the to-do business logic.
package service

import "errors"

// ErrNotFound is returned when a task ID does not exist.
var ErrNotFound = errors.New("task not found")

// Task is a single to-do item.
type Task struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
}
