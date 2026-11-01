# Todo App

## 1. Requirements & scope

**Goal:** a small single-user to-do list web app that runs locally.

**In scope**
- Add, edit, delete tasks.
- Mark a task complete / incomplete.
- Persistence: tasks and their states survive a restart.
- One HTML page served by the Go app (plain HTML/CSS/vanilla JS, no build step).

**Out of scope:** auth / multi-user, due dates, priorities, tags, search, sync, deployment/Docker, frontend frameworks or build toolchain, pagination, ordering controls.

**Validation:** title is trimmed; must be non-empty; max 200 characters (counted in runes). Unknown ID -> 404. Bad input / malformed JSON -> 400. Never panic.

**Acceptance criteria:** as listed in the task brief (add, edit, empty title rejected, delete, toggle, restart persistence, proper errors, table-driven testify tests + lint pass, persistence-across-restart test).

**Decisions (confirmed by user)**
1. **Go 1.22.12**, installed to `~/sdk/go1.22.12` (checksum verified against go.dev; system Go 1.15 untouched). `go.mod` says `go 1.22`; the Makefile uses `GO ?= $(HOME)/sdk/go1.22.12/bin/go`. Reasoning: Go 1.15 has no `//go:embed`, so the page would be read from disk and the binary only works from the repo root; it also lacks `os.ReadFile`/`io.ReadAll`, and Go 1.22 gives `http.ServeMux` method+path patterns (`PATCH /api/tasks/{id}/toggle`), so no manual path parsing. testify and golangci-lint also target modern Go. Cost: one 70 MB local download, no repo impact.
2. **Lint is low priority.** `make lint` = `gofmt -l` + `go vet`; golangci-lint is used only if it is already installed or installs cleanly, otherwise skipped (and reported).
3. **testify** (`assert`/`require`) is approved; unit tests are required.
4. **Persistence: SQLite** (option A below, replaces the earlier JSON-file recommendation). Reason: a JSON file cannot serve multiple users or instances (whole-file rewrite, process-local mutex, no queries), SQLite keeps zero-setup local running and can move to Postgres later by swapping the repository. Approved dependency: `modernc.org/sqlite` (pure Go, no cgo). Estimated extra cost ~10-20 min.
5. **Focus on the three main features first**; UX polish and extra features come after they work.

## 2. Technical design

**Layout**
```
cmd/todo/main.go          wiring, flags (-addr, -data), graceful shutdown
internal/handler/         HTTP routes, JSON in/out, error mapping
internal/service/         validation + business logic, Task type, sentinel errors
internal/repository/      SQLite store (database/sql)
web/index.html            single page (inline CSS + JS)
Makefile  README.md  prompt.md  reflection.md  docs/  screenshots/
```
Dependency direction: handler -> service -> repository. The service defines the small `Repository` interface it consumes (needed anyway so service tests can use a fake).

**API**
| Method | Path | Body | Success | Errors |
|---|---|---|---|---|
| GET | /api/tasks | - | 200 list | - |
| POST | /api/tasks | `{"title"}` | 201 task | 400 |
| PUT | /api/tasks/{id} | `{"title"}` | 200 task | 400, 404 |
| PATCH | /api/tasks/{id}/toggle | - | 200 task | 404 |
| DELETE | /api/tasks/{id} | - | 204 | 404 |

Errors are `{"error":"message"}`; internal errors return a generic 500 message and are logged at the handler. Routing uses Go 1.22 `net/http` `ServeMux` method+path patterns (no router dependency). `web/index.html` is embedded with `//go:embed`, so the binary is self-contained. Request bodies are capped (`http.MaxBytesReader`, 1 KB).

**Persistence options**

| | A. SQLite (`database/sql`) | B. JSON file |
|---|---|---|
| Multi-user later | Yes (add `user_id` column) | No: whole-file rewrite, process-local mutex |
| Concurrent writers | One at a time (enough here) | Serialized by mutex, single process only |
| Setup | None (a file), 1 dependency | None, no dependency |
| Scale path | Swap to Postgres behind the same repository | Rewrite persistence |

**Chosen: A (SQLite).** Driver `modernc.org/sqlite` (pure Go). Default DB path `./data/todo.db`, override with `-db`. One table created at startup with `CREATE TABLE IF NOT EXISTS tasks (id INTEGER PRIMARY KEY AUTOINCREMENT, title TEXT NOT NULL, completed INTEGER NOT NULL DEFAULT 0, created_at TEXT NOT NULL)`. `AUTOINCREMENT` so IDs are not reused after delete. `SetMaxOpenConns(1)` plus a busy timeout to avoid "database is locked". Plain SQL, no ORM, no migration framework. No `user_id` column yet (multi-user is out of scope; it is a one-line migration later).

**Risks / notes**
- SQLite is one machine / one writer at a time; multiple app instances need Postgres (out of scope).
- Each mutation is a single SQL statement, so it is atomic; no in-memory copy, so memory and disk cannot diverge.
- Corrupt or unreadable DB file is a startup error.
- Double-click on toggle simply toggles twice; no idempotency key (single user, out of scope).

**Steps (each ends with test + lint, commit, push if a remote exists)**
1. Scaffold: go.mod, Makefile, main skeleton, `.gitignore`, repository (SQLite) + tests.
2. Service (validation, errors) + tests.
3. Handler (API) + tests with `httptest`.
4. Web page + wiring in `main.go`; manual run verified.
5. Screenshots (headless browser as dev-only tool), README, reflection.md.

## 3. Test plan

Table-driven, testify `assert`/`require`; run with `make test` (`go test -race ./...`).

**Repository** (temp dir DB): create/list/update/delete/toggle; fresh DB -> empty list; non-database file -> open error; unknown ID -> `ErrNotFound`; IDs not reused after delete; **restart test**: write via store 1, close it, open store 2 on the same file, assert identical tasks and completed states.

**Service** (fake repository): title trim; empty / whitespace-only rejected; exactly 200 runes OK, 201 rejected; multi-byte title counted by runes; unknown ID on edit/toggle/delete -> `ErrNotFound`; repository error is wrapped and propagated.

**Handler** (`httptest` + real service on a temp-DB repo): each endpoint's happy path and status code; empty title -> 400; malformed JSON -> 400; oversized body -> 400; non-numeric / unknown ID -> 400 / 404; wrong method -> 405; error body shape `{"error":...}`.

**Manual / end-to-end:** run the real binary, exercise add/edit/toggle/delete in the browser, restart, confirm data persists; screenshots come from this run.

**Mocks:** only the repository interface in service tests; no other external dependencies.
