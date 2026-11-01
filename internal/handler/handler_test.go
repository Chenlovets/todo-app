package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"todoapp/internal/repository"
	"todoapp/internal/service"
)

func newServer(t *testing.T) http.Handler {
	t.Helper()
	repo, err := repository.Open(filepath.Join(t.TempDir(), "todo.db"))
	require.NoError(t, err)
	t.Cleanup(func() { repo.Close() })
	return New(service.New(repo))
}

func do(h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func decodeTask(t *testing.T, rec *httptest.ResponseRecorder) service.Task {
	t.Helper()
	var task service.Task
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &task))
	return task
}

func TestHappyPath(t *testing.T) {
	h := newServer(t)

	// empty list is [] not null
	rec := do(h, "GET", "/api/tasks", "")
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `[]`, rec.Body.String())

	rec = do(h, "POST", "/api/tasks", `{"title":"  buy milk "}`)
	require.Equal(t, http.StatusCreated, rec.Code)
	task := decodeTask(t, rec)
	assert.Equal(t, "buy milk", task.Title)
	assert.False(t, task.Completed)

	rec = do(h, "PUT", "/api/tasks/1", `{"title":"buy oat milk"}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "buy oat milk", decodeTask(t, rec).Title)

	rec = do(h, "PATCH", "/api/tasks/1/toggle", "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, decodeTask(t, rec).Completed)
	rec = do(h, "PATCH", "/api/tasks/1/toggle", "")
	assert.False(t, decodeTask(t, rec).Completed)

	rec = do(h, "GET", "/api/tasks", "")
	assert.JSONEq(t, `[{"id":1,"title":"buy oat milk","completed":false}]`, rec.Body.String())

	rec = do(h, "DELETE", "/api/tasks/1", "")
	assert.Equal(t, http.StatusNoContent, rec.Code)
	rec = do(h, "GET", "/api/tasks", "")
	assert.JSONEq(t, `[]`, rec.Body.String())
}

func TestErrors(t *testing.T) {
	tests := []struct {
		name, method, path, body string
		wantStatus               int
		wantErr                  string // substring of the "error" field; empty = skip
	}{
		{"add empty title", "POST", "/api/tasks", `{"title":""}`, 400, "title"},
		{"add whitespace title", "POST", "/api/tasks", `{"title":"   "}`, 400, "title"},
		{"add too long", "POST", "/api/tasks", `{"title":"` + strings.Repeat("a", 201) + `"}`, 400, "200"},
		{"add malformed json", "POST", "/api/tasks", `{"title":`, 400, "JSON"},
		{"add empty body", "POST", "/api/tasks", ``, 400, "JSON"},
		{"add wrong type", "POST", "/api/tasks", `{"title":123}`, 400, "JSON"},
		{"add oversized body", "POST", "/api/tasks", `{"title":"` + strings.Repeat("a", 2000) + `"}`, 400, "JSON"},
		{"edit unknown id", "PUT", "/api/tasks/999", `{"title":"x"}`, 404, "not found"},
		{"edit non-numeric id", "PUT", "/api/tasks/abc", `{"title":"x"}`, 400, "id"},
		{"edit zero id", "PUT", "/api/tasks/0", `{"title":"x"}`, 400, "id"},
		{"edit negative id", "PUT", "/api/tasks/-1", `{"title":"x"}`, 400, "id"},
		{"edit overflow id", "PUT", "/api/tasks/99999999999999999999", `{"title":"x"}`, 400, "id"},
		{"toggle unknown id", "PATCH", "/api/tasks/999/toggle", "", 404, "not found"},
		{"toggle bad id", "PATCH", "/api/tasks/abc/toggle", "", 400, "id"},
		{"delete unknown id", "DELETE", "/api/tasks/999", "", 404, "not found"},
		{"delete bad id", "DELETE", "/api/tasks/abc", "", 400, "id"},
		{"wrong method", "POST", "/api/tasks/1", `{}`, 405, ""},
		{"unknown route", "GET", "/api/nope", "", 404, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newServer(t)
			rec := do(h, tt.method, tt.path, tt.body)
			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantErr != "" {
				var body map[string]string
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
				assert.Contains(t, body["error"], tt.wantErr)
			}
		})
	}
}

func TestRejectedEditKeepsTitle(t *testing.T) {
	h := newServer(t)
	do(h, "POST", "/api/tasks", `{"title":"keep me"}`)

	rec := do(h, "PUT", "/api/tasks/1", `{"title":"  "}`)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	rec = do(h, "GET", "/api/tasks", "")
	assert.JSONEq(t, `[{"id":1,"title":"keep me","completed":false}]`, rec.Body.String())
}

func TestInternalErrorHidesDetails(t *testing.T) {
	repo, err := repository.Open(filepath.Join(t.TempDir(), "todo.db"))
	require.NoError(t, err)
	h := New(service.New(repo))
	require.NoError(t, repo.Close()) // force DB errors

	rec := do(h, "GET", "/api/tasks", "")
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.JSONEq(t, `{"error":"internal server error"}`, rec.Body.String())
}
