package repository

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"todoapp/internal/service"
)

func newStore(t *testing.T) (*SQLite, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "todo.db")
	s, err := Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { s.Close() })
	return s, path
}

func TestCreateAndList(t *testing.T) {
	ctx := context.Background()
	s, _ := newStore(t)

	got, err := s.List(ctx)
	require.NoError(t, err)
	assert.Empty(t, got, "fresh DB has no tasks")

	a, err := s.Create(ctx, "a")
	require.NoError(t, err)
	b, err := s.Create(ctx, "b")
	require.NoError(t, err)
	assert.Equal(t, service.Task{ID: a.ID, Title: "a"}, a)

	got, err = s.List(ctx)
	require.NoError(t, err)
	assert.Equal(t, []service.Task{a, b}, got)
}

func TestModify(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name  string
		op    func(s *SQLite, id int64) error
		check func(t *testing.T, got service.Task)
	}{
		{"update title",
			func(s *SQLite, id int64) error { return s.UpdateTitle(ctx, id, "new") },
			func(t *testing.T, got service.Task) { assert.Equal(t, "new", got.Title) }},
		{"set completed",
			func(s *SQLite, id int64) error { return s.SetCompleted(ctx, id, true) },
			func(t *testing.T, got service.Task) { assert.True(t, got.Completed) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, _ := newStore(t)
			task, err := s.Create(ctx, "old")
			require.NoError(t, err)

			require.NoError(t, tt.op(s, task.ID))
			got, err := s.Get(ctx, task.ID)
			require.NoError(t, err)
			tt.check(t, got)

			// Unknown ID must be ErrNotFound.
			assert.ErrorIs(t, tt.op(s, task.ID+999), service.ErrNotFound)
		})
	}
}

func TestSetCompletedBackToFalse(t *testing.T) {
	ctx := context.Background()
	s, _ := newStore(t)
	task, err := s.Create(ctx, "x")
	require.NoError(t, err)
	require.NoError(t, s.SetCompleted(ctx, task.ID, true))
	require.NoError(t, s.SetCompleted(ctx, task.ID, false))
	got, err := s.Get(ctx, task.ID)
	require.NoError(t, err)
	assert.False(t, got.Completed)
}

func TestDelete(t *testing.T) {
	ctx := context.Background()
	s, _ := newStore(t)
	a, _ := s.Create(ctx, "a")

	require.NoError(t, s.Delete(ctx, a.ID))
	_, err := s.Get(ctx, a.ID)
	assert.ErrorIs(t, err, service.ErrNotFound)
	assert.ErrorIs(t, s.Delete(ctx, a.ID), service.ErrNotFound, "second delete")

	// AUTOINCREMENT: IDs are not reused after delete.
	b, err := s.Create(ctx, "b")
	require.NoError(t, err)
	assert.Greater(t, b.ID, a.ID)
}

// TestHostileTitleStoredLiterally checks that SQL metacharacters are data, not code.
func TestHostileTitleStoredLiterally(t *testing.T) {
	ctx := context.Background()
	s, _ := newStore(t)
	payload := `x'); DROP TABLE tasks;--`

	a, err := s.Create(ctx, payload)
	require.NoError(t, err)
	require.NoError(t, s.UpdateTitle(ctx, a.ID, payload+"2"))

	got, err := s.List(ctx)
	require.NoError(t, err, "tasks table must still exist")
	require.Len(t, got, 1)
	assert.Equal(t, payload+"2", got[0].Title)
}

func TestGetUnknown(t *testing.T) {
	s, _ := newStore(t)
	_, err := s.Get(context.Background(), 42)
	assert.ErrorIs(t, err, service.ErrNotFound)
}

// TestPersistsAcrossRestart simulates an app restart: close the store, reopen the same file.
func TestPersistsAcrossRestart(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "todo.db")

	s1, err := Open(path)
	require.NoError(t, err)
	a, _ := s1.Create(ctx, "buy milk")
	b, _ := s1.Create(ctx, "write code")
	require.NoError(t, s1.SetCompleted(ctx, b.ID, true))
	require.NoError(t, s1.UpdateTitle(ctx, a.ID, "buy oat milk"))
	c, _ := s1.Create(ctx, "to delete")
	require.NoError(t, s1.Delete(ctx, c.ID))
	require.NoError(t, s1.Close())

	s2, err := Open(path)
	require.NoError(t, err)
	defer s2.Close()
	got, err := s2.List(ctx)
	require.NoError(t, err)
	assert.Equal(t, []service.Task{
		{ID: a.ID, Title: "buy oat milk", Completed: false},
		{ID: b.ID, Title: "write code", Completed: true},
	}, got)
}

func TestOpenErrors(t *testing.T) {
	t.Run("not a database file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "bad.db")
		require.NoError(t, os.WriteFile(path, []byte("this is definitely not a sqlite database file........"), 0o600))
		_, err := Open(path)
		assert.Error(t, err)
	})
	t.Run("missing directory", func(t *testing.T) {
		_, err := Open(filepath.Join(t.TempDir(), "nope", "todo.db"))
		assert.Error(t, err)
	})
}
