package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// MaxTitleLen is the maximum title length in characters (runes).
const MaxTitleLen = 200

// ErrInvalidTitle is returned when a title is empty or too long.
var ErrInvalidTitle = errors.New("invalid title")

// Repository is the storage the service needs.
type Repository interface {
	List(ctx context.Context) ([]Task, error)
	Create(ctx context.Context, title string) (Task, error)
	Get(ctx context.Context, id int64) (Task, error)
	UpdateTitle(ctx context.Context, id int64, title string) error
	SetCompleted(ctx context.Context, id int64, completed bool) error
	Delete(ctx context.Context, id int64) error
}

// Service implements the to-do use cases.
type Service struct {
	repo Repository
}

// New creates a Service.
func New(repo Repository) *Service { return &Service{repo: repo} }

// validateTitle trims the title and checks it before it is stored: non-empty, valid UTF-8,
// no control characters (newlines, NUL, ...), and at most MaxTitleLen characters.
func validateTitle(title string) (string, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return "", fmt.Errorf("%w: must not be empty", ErrInvalidTitle)
	}
	if !utf8.ValidString(title) {
		return "", fmt.Errorf("%w: must be valid UTF-8", ErrInvalidTitle)
	}
	if strings.IndexFunc(title, unicode.IsControl) >= 0 {
		return "", fmt.Errorf("%w: must not contain control characters", ErrInvalidTitle)
	}
	if utf8.RuneCountInString(title) > MaxTitleLen {
		return "", fmt.Errorf("%w: must be at most %d characters", ErrInvalidTitle, MaxTitleLen)
	}
	return title, nil
}

// List returns all tasks.
func (s *Service) List(ctx context.Context) ([]Task, error) {
	return s.repo.List(ctx)
}

// Add creates a task with the given title.
func (s *Service) Add(ctx context.Context, title string) (Task, error) {
	title, err := validateTitle(title)
	if err != nil {
		return Task{}, err
	}
	return s.repo.Create(ctx, title)
}

// Edit changes a task's title and returns the updated task.
func (s *Service) Edit(ctx context.Context, id int64, title string) (Task, error) {
	title, err := validateTitle(title)
	if err != nil {
		return Task{}, err
	}
	if err := s.repo.UpdateTitle(ctx, id, title); err != nil {
		return Task{}, fmt.Errorf("edit task: %w", err)
	}
	return s.repo.Get(ctx, id)
}

// Toggle flips a task's completed state and returns the updated task.
func (s *Service) Toggle(ctx context.Context, id int64) (Task, error) {
	t, err := s.repo.Get(ctx, id)
	if err != nil {
		return Task{}, fmt.Errorf("toggle task: %w", err)
	}
	if err := s.repo.SetCompleted(ctx, id, !t.Completed); err != nil {
		return Task{}, fmt.Errorf("toggle task: %w", err)
	}
	t.Completed = !t.Completed
	return t, nil
}

// Delete removes a task.
func (s *Service) Delete(ctx context.Context, id int64) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	return nil
}
