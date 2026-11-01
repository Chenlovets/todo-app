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
	return doCT(h, method, path, body, "application/json")
}

func doCT(h http.Handler, method, path, body, contentType string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
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
		{"add oversized body", "POST", "/api/tasks", `{"title":"` + strings.Repeat("a", maxBodyBytes) + `"}`, 400, "too large"},
		{"add trailing garbage", "POST", "/api/tasks", `{"title":"x"} garbage`, 400, "JSON"},
		{"add second json value", "POST", "/api/tasks", `{"title":"x"}{"title":"y"}`, 400, "JSON"},
		{"add unknown field", "POST", "/api/tasks", `{"title":"x","completed":true}`, 400, "JSON"},
		{"add control char escaped", "POST", "/api/tasks", `{"title":"a\u0000b"}`, 400, "control"},
		{"add newline escaped", "POST", "/api/tasks", `{"title":"a\nb"}`, 400, "control"},
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

func TestContentType(t *testing.T) {
	tests := []struct {
		name, contentType string
		wantStatus        int
	}{
		{"json", "application/json", 201},
		{"json with charset", "application/json; charset=utf-8", 201},
		{"missing", "", 415},
		{"text/plain (cross-site form trick)", "text/plain", 415},
		{"form", "application/x-www-form-urlencoded", 415},
		{"garbage", "a/b/c;;", 415},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newServer(t)
			rec := doCT(h, "POST", "/api/tasks", `{"title":"x"}`, tt.contentType)
			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantStatus == 415 {
				rec = do(h, "GET", "/api/tasks", "")
				assert.JSONEq(t, `[]`, rec.Body.String(), "rejected request must not create a task")
			}
		})
	}
	// PUT is checked the same way.
	h := newServer(t)
	do(h, "POST", "/api/tasks", `{"title":"keep"}`)
	assert.Equal(t, 415, doCT(h, "PUT", "/api/tasks/1", `{"title":"changed"}`, "text/plain").Code)
}

func TestBodyLimitFitsMaxTitle(t *testing.T) {
	h := newServer(t)
	// 200 characters, each sent as a 6-byte \u escape (1200 bytes) and as 4-byte emoji must be accepted.
	escaped := strings.Repeat(`\u00e9`, service.MaxTitleLen)
	rec := do(h, "POST", "/api/tasks", `{"title":"`+escaped+`"}`)
	assert.Equal(t, http.StatusCreated, rec.Code)
	rec = do(h, "POST", "/api/tasks", `{"title":"`+strings.Repeat("😀", service.MaxTitleLen)+`"}`)
	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestHostileTitleRoundTrip(t *testing.T) {
	h := newServer(t)
	for _, title := range []string{`x'); DROP TABLE tasks;--`, `<script>alert(1)</script>`, `" OR 1=1 --`} {
		body, _ := json.Marshal(map[string]string{"title": title})
		rec := do(h, "POST", "/api/tasks", string(body))
		require.Equal(t, http.StatusCreated, rec.Code)
		assert.Equal(t, title, decodeTask(t, rec).Title)
	}
	rec := do(h, "GET", "/api/tasks", "")
	assert.Equal(t, http.StatusOK, rec.Code, "table must survive")
	assert.Contains(t, rec.Body.String(), "DROP TABLE")
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
