// Package handler exposes the to-do service over HTTP.
package handler

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"mime"
	"net/http"
	"strconv"

	"todoapp/internal/service"
)

// maxBodyBytes fits a 200-character title even if a client \u-escapes every character.
const maxBodyBytes = 4 << 10

// Handler serves the JSON API.
type Handler struct {
	svc *service.Service
}

// New returns an http.Handler with the API routes registered.
func New(svc *service.Service) http.Handler {
	h := &Handler{svc: svc}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/tasks", h.list)
	mux.HandleFunc("POST /api/tasks", h.add)
	mux.HandleFunc("PUT /api/tasks/{id}", h.edit)
	mux.HandleFunc("PATCH /api/tasks/{id}/toggle", h.toggle)
	mux.HandleFunc("DELETE /api/tasks/{id}", h.delete)
	return mux
}

type titleRequest struct {
	Title string `json:"title"`
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	tasks, err := h.svc.List(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, tasks)
}

func (h *Handler) add(w http.ResponseWriter, r *http.Request) {
	var req titleRequest
	if !decodeBody(w, r, &req) {
		return
	}
	t, err := h.svc.Add(r.Context(), req.Title)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

func (h *Handler) edit(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var req titleRequest
	if !decodeBody(w, r, &req) {
		return
	}
	t, err := h.svc.Edit(r.Context(), id, req.Title)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (h *Handler) toggle(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	t, err := h.svc.Toggle(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := h.svc.Delete(r.Context(), id); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func parseID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid task id"})
		return 0, false
	}
	return id, true
}

// decodeBody reads a JSON request body strictly: Content-Type must be application/json (this also
// blocks cross-site form posts), size is capped, unknown fields and trailing data are rejected.
// On failure it writes an error response and returns false.
func decodeBody(w http.ResponseWriter, r *http.Request, v any) bool {
	if mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err != nil || mt != "application/json" {
		writeJSON(w, http.StatusUnsupportedMediaType, map[string]string{"error": "Content-Type must be application/json"})
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	err := dec.Decode(v)
	if err == nil {
		if _, err = dec.Token(); errors.Is(err, io.EOF) {
			return true
		}
	}
	msg := "invalid JSON body"
	var tooBig *http.MaxBytesError
	if errors.As(err, &tooBig) {
		msg = "request body too large"
	}
	writeJSON(w, http.StatusBadRequest, map[string]string{"error": msg})
	return false
}

// writeError maps service errors to HTTP statuses; internal details are logged, not returned.
func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "task not found"})
	case errors.Is(err, service.ErrInvalidTitle):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
	default:
		log.Printf("internal error: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write response: %v", err)
	}
}
