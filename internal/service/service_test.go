package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errBoom = errors.New("boom")

// fakeRepo is an in-memory Repository; fail makes every call return errBoom.
type fakeRepo struct {
	tasks  map[int64]Task
	nextID int64
	fail   bool
}

func newFake() *fakeRepo { return &fakeRepo{tasks: map[int64]Task{}, nextID: 1} }

func (f *fakeRepo) List(context.Context) ([]Task, error) {
	if f.fail {
		return nil, errBoom
	}
	out := []Task{}
	for id := int64(1); id < f.nextID; id++ {
		if t, ok := f.tasks[id]; ok {
			out = append(out, t)
		}
	}
	return out, nil
}

func (f *fakeRepo) Create(_ context.Context, title string) (Task, error) {
	if f.fail {
		return Task{}, errBoom
	}
	t := Task{ID: f.nextID, Title: title}
	f.tasks[t.ID] = t
	f.nextID++
	return t, nil
}

func (f *fakeRepo) Get(_ context.Context, id int64) (Task, error) {
	if f.fail {
		return Task{}, errBoom
	}
	t, ok := f.tasks[id]
	if !ok {
		return Task{}, ErrNotFound
	}
	return t, nil
}

func (f *fakeRepo) UpdateTitle(_ context.Context, id int64, title string) error {
	if f.fail {
		return errBoom
	}
	t, ok := f.tasks[id]
	if !ok {
		return ErrNotFound
	}
	t.Title = title
	f.tasks[id] = t
	return nil
}

func (f *fakeRepo) SetCompleted(_ context.Context, id int64, completed bool) error {
	if f.fail {
		return errBoom
	}
	t, ok := f.tasks[id]
	if !ok {
		return ErrNotFound
	}
	t.Completed = completed
	f.tasks[id] = t
	return nil
}

func (f *fakeRepo) Delete(_ context.Context, id int64) error {
	if f.fail {
		return errBoom
	}
	if _, ok := f.tasks[id]; !ok {
		return ErrNotFound
	}
	delete(f.tasks, id)
	return nil
}

func TestAddAndEditTitleValidation(t *testing.T) {
	tests := []struct {
		name      string
		title     string
		want      string
		wantInval bool
	}{
		{"normal", "buy milk", "buy milk", false},
		{"trimmed", "  buy milk \t\n", "buy milk", false},
		{"empty", "", "", true},
		{"whitespace only", "  \t\n ", "", true},
		{"exactly max length", strings.Repeat("a", MaxTitleLen), strings.Repeat("a", MaxTitleLen), false},
		{"one over max", strings.Repeat("a", MaxTitleLen+1), "", true},
		{"multibyte counted as runes", strings.Repeat("任", MaxTitleLen), strings.Repeat("任", MaxTitleLen), false},
		{"multibyte one over", strings.Repeat("任", MaxTitleLen+1), "", true},
		{"newline inside", "line1\nline2", "", true},
		{"tab inside", "a\tb", "", true},
		{"NUL byte", "a\x00b", "", true},
		{"DEL char", "a\x7fb", "", true},
		{"C1 control char", "a\u0085b", "", true},
		{"invalid UTF-8", "a\xffb", "", true},
		{"SQL-looking text is plain text", "x'); DROP TABLE tasks;--", "x'); DROP TABLE tasks;--", false},
		{"HTML-looking text is plain text", "<script>alert(1)</script>", "<script>alert(1)</script>", false},
		{"padding does not count", " " + strings.Repeat("a", MaxTitleLen) + " ", strings.Repeat("a", MaxTitleLen), false},
	}
	for _, tt := range tests {
		t.Run("add/"+tt.name, func(t *testing.T) {
			svc := New(newFake())
			got, err := svc.Add(context.Background(), tt.title)
			if tt.wantInval {
				assert.ErrorIs(t, err, ErrInvalidTitle)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got.Title)
			assert.False(t, got.Completed)
		})
		t.Run("edit/"+tt.name, func(t *testing.T) {
			svc := New(newFake())
			task, err := svc.Add(context.Background(), "orig")
			require.NoError(t, err)
			got, err := svc.Edit(context.Background(), task.ID, tt.title)
			if tt.wantInval {
				assert.ErrorIs(t, err, ErrInvalidTitle)
				stored, _ := svc.List(context.Background())
				assert.Equal(t, "orig", stored[0].Title, "rejected edit must not change the task")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got.Title)
		})
	}
}

func TestToggle(t *testing.T) {
	ctx := context.Background()
	svc := New(newFake())
	task, err := svc.Add(ctx, "x")
	require.NoError(t, err)

	got, err := svc.Toggle(ctx, task.ID)
	require.NoError(t, err)
	assert.True(t, got.Completed)

	got, err = svc.Toggle(ctx, task.ID)
	require.NoError(t, err)
	assert.False(t, got.Completed, "second toggle un-completes")
}

func TestUnknownID(t *testing.T) {
	ctx := context.Background()
	svc := New(newFake())
	tests := []struct {
		name string
		call func() error
	}{
		{"edit", func() error { _, err := svc.Edit(ctx, 99, "t"); return err }},
		{"toggle", func() error { _, err := svc.Toggle(ctx, 99); return err }},
		{"delete", func() error { return svc.Delete(ctx, 99) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.ErrorIs(t, tt.call(), ErrNotFound)
		})
	}
}

func TestDelete(t *testing.T) {
	ctx := context.Background()
	svc := New(newFake())
	a, _ := svc.Add(ctx, "a")
	b, _ := svc.Add(ctx, "b")

	require.NoError(t, svc.Delete(ctx, a.ID))
	got, err := svc.List(ctx)
	require.NoError(t, err)
	assert.Equal(t, []Task{b}, got)
}

func TestRepositoryErrorsPropagate(t *testing.T) {
	ctx := context.Background()
	repo := newFake()
	svc := New(repo)
	task, err := svc.Add(ctx, "a")
	require.NoError(t, err)
	repo.fail = true

	tests := []struct {
		name string
		call func() error
	}{
		{"list", func() error { _, err := svc.List(ctx); return err }},
		{"add", func() error { _, err := svc.Add(ctx, "b"); return err }},
		{"edit", func() error { _, err := svc.Edit(ctx, task.ID, "b"); return err }},
		{"toggle", func() error { _, err := svc.Toggle(ctx, task.ID); return err }},
		{"delete", func() error { return svc.Delete(ctx, task.ID) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call()
			assert.ErrorIs(t, err, errBoom)
			assert.NotErrorIs(t, err, ErrNotFound)
		})
	}
}
