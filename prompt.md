# Prompts

## Prompt 1 - 2026-11-01 14:18

```
Build a to-do list web app, following the workflow in my CLAUDE.md (sections 1, 8, 9 and 10).

## Phase 1: Plan only (do NOT write app code yet)
1. Create prompt.md in the repo root and append this message verbatim (see Deliverable 1).
2. Write docs/todo-app.md with three sections: Requirements & scope, Technical design, Test plan.
3. Ask me only about points that are genuinely ambiguous.
4. Stop and wait for my confirmation of the docs before writing any app code.

## Goal
A small single-user to-do list app that runs locally.

## Main features
1. Add, edit, and delete tasks
2. Mark a task as complete (and un-complete it)
3. Basic persistence: tasks and their states survive an app restart

## Out of scope (do not build)
Auth or multi-user, due dates, priorities, tags, search, sync, deployment/Docker, any frontend framework or frontend build toolchain.

## Constraints
- Go backend. Keep dependencies minimal: standard library first, add a library only with a clear reason.
- Persistence: propose 2 options (e.g. JSON file vs SQLite) with trade-offs in the design doc, and recommend one.
- UI: one simple page served by the Go app (plain HTML/CSS/vanilla JS, no build step).
- Keep layering light (handler -> service -> repository). No speculative abstractions.
- Validation: titles are trimmed, non-empty, with a sensible max length. Unknown IDs and bad input return clear errors, never a crash.
- Add a Makefile with build, test, lint, run targets (CLAUDE.md section 7).

## Acceptance criteria
- I can add a task and see it in the list.
- Editing changes the title; empty titles are rejected.
- Deleting removes the task.
- Completing a task toggles its state, visible in the UI.
- After restarting the app, all tasks and states are intact.
- Invalid input and unknown IDs return proper errors.
- Tests (table-driven + testify) and lint pass, including a persistence-across-restart test.

## Deliverables (all committed to the repo)
1. prompt.md (repo root): append every message I send you, verbatim, unedited and unsummarized, in order, starting with this one. Format: "## Prompt N - <timestamp>" followed by the raw text in a fenced block. Do this at the start of each turn, before any other work.
2. README.md: how to run the app (exact commands, verified by actually running them), a step-by-step guide for each main feature (add, edit, delete, complete, persistence), where data is stored, how to run tests, and the screenshots embedded.
3. screenshots/ folder: after development and tests pass, run the real app and capture its interface and main features (empty state, tasks added, editing, completed, after delete, still there after restart). Name them 01-empty.png, 02-added.png, etc. Use a headless browser; installing it as a dev-only tool is pre-approved, but it must not become a runtime dependency of the app. If you cannot capture real screenshots in this environment, tell me explicitly and give me manual instructions. Never fake or mock them.
4. reflection.md: honest and specific, based only on what actually happened in this session. Cover (a) a summary of our discussion and key decisions, (b) what you got wrong initially and how it was fixed, (c) what I could have done better in my prompts and answers, and what you could have done better, (d) concrete suggestions to make our workflow, or my CLAUDE.md, more efficient. If nothing went wrong in an area, say so instead of padding.
   To make (b) accurate: in every step record in .dev-log, add a line "Mistakes / rework" listing wrong assumptions, misreadings of my instructions, failed tests or rework in that step.

## Execution rules
- Git: if this is not a repo yet, git init, make an initial commit on main, then work on feat/todo-app. Commit and push each step per section 8/9. If no remote is configured, commit locally and skip the push without asking; tell me at the end.
- prompt.md, README.md, reflection.md, docs/ and screenshots/ are deliverables and get committed. .dev-log/ stays local.
- Order of work: steps from the confirmed plan -> all tests and lint pass -> screenshots -> final README with screenshots -> reflection.md -> final commit and a short summary to me.
- Pause only for the conditions in section 9.3.
```

## Prompt 2 - 2026-11-01 14:26

```
1. update go version if you think it would be more effieient for you moving forward (record your reasoning if you think so) 2. lint is a minor thing, do what you think is more effiecient, focus on the main features. 3. yes, unit test is needed. also do delete the first prompt that is repeated. Do i answer all your questions?
```

## Prompt 3 - 2026-11-01 14:33

```
regarding your presistence stroge choice, does it also work for multiple users? if not, think of another solution that is easier for the app to scale up in the future
```

## Prompt 4 - 2026-11-01 14:36

```
to make the decision, i need your estimation of how much more time you need to take for A
```

## Prompt 5 - 2026-11-01 14:38

```
Go with A then. Remember do focus on the three main features, we can make optimization or add more features for better user experience after that!
```

## Prompt 6 - 2026-11-01 14:42

```
yes I approve
```

## Prompt 7 - 2026-11-01 14:46

```
you dont need to record your reasoning. continue the work
```
