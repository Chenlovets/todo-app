# AI Usage Write-up Answer

1. Before prompting, i list the required features, deliverables, requirements and constraints, then ask my claude agent to initiaite a raw prompt structure to me based on my local CLAUDE.md file, which records my working habit and genral guidence for development work.
2. The AI misidentified some risk and overlooked some testing issue, i ask it to start a seperate review and testing agents to help.
3. I did not deleget review and test to the main agent itself, but used subagent instead because it's better with a clean slate. Also i test the interfaces myself multiple times to give AI an actual human test feedback and added a few optimiztion features.
4. I encountered some API error regarding reasoning extraction a few times becasue i tell AI to record thought process before starting to code, which triggered the error. I tried to delete the promot in the polluted session but it still camp up. So I restart a new session and let AI starts clean slate to code based our discussed plan, which is what i should've done in the first place. Also a throughout code review and assessment with another agent like codex could benefit with more time.

---

# Reflection

Based only on what happened in this project. Sources: `prompt.md` (your 24 prompts), `.dev-log/todo-app.md` (my per-step notes) and the work after the last context reset. I have no record of my own replies to prompts 1-8 beyond what the dev-log and docs say, so (a) and (b) for that stretch lean on those.

## (a) Discussion and key decisions

- **Plan first.** Prompt 1 asked for docs only, no code, then your confirmation. I wrote `docs/todo-app.md` (requirements, design, test plan) and you approved it (prompt 6).
- **Go version.** The machine has Go 1.15. You let me upgrade if it helped (prompt 2). I installed 1.22.12 locally (checksum verified, system Go untouched) for `//go:embed`, `ServeMux` method+path routing and a testify-compatible toolchain.
- **Lint is minor, tests are required** (prompt 2). `make lint` is `gofmt` + `go vet`; golangci-lint isn't installed and was skipped each time.
- **Persistence.** I first recommended a JSON file. You asked whether it works for multiple users (prompt 3). It doesn't, so the doc switched to SQLite (`modernc.org/sqlite`, pure Go). You asked for a time estimate (prompt 4) and chose it (prompt 5).
- **Steps.** 1 repository, 2 service, 3 handler, 4 web page and wiring, 5 screenshots/README/reflection. Each step committed locally; no remote exists so nothing was pushed.
- **After you tested it yourself** (prompt 13): the page didn't refresh across windows. We chose focus/visibility reload plus 5s polling over SSE.
- **Independent checks** (prompts 17-18): a no-context subagent followed the README (first pass only drove the API; a stricter pass used `make run` and a real browser and passed), and a second subagent reviewed the code against the design doc.
- **Hardening** (prompt 19): the review found real bugs and you asked for input safety. We fixed shutdown, the stuck edit mode, lost edit text, wrong docs, and added title/Content-Type/JSON checks. The wrong-method 405 stays plain text by your decision (prompt 22).

## (b) What I got wrong, and the fix

Step 1-4 items are copied from the dev-log; the rest I found in step 5 and afterwards.

1. **Assumed modern tooling on an old machine** (git 2.21, Go 1.15): `git init -b` and `go install pkg@version` failed; worked around.
2. **Dependency versions.** `go get` failed on a missing GOSUMDB (GOPROXY is goproxy.cn), then I took the newest sqlite, which needs Go 1.26. Re-pinned to sqlite v1.34.5 / testify v1.9.0.
3. **Shell `GOROOT` pointed at Go 1.15** and broke Go 1.22 builds. The Makefile now forces `GOROOT`. I hit the same thing again later by calling `go` directly instead of `make`.
4. **Wrong page location.** I planned `web/index.html` at the repo root, which `go:embed` can't reach. Moved it to `cmd/todo/web` and fixed the doc.
5. **Malformed tool calls**: one test-file write was rejected and rewritten; one dev-log heredoc was cut off and redone. No test failures in steps 1-4.
6. **A real UI bug shipped as "done" in step 4.** `load()` cleared the error message right after `run()` showed it, so "title must not be empty" flashed and vanished. I had written that the browser behaviour wasn't exercised yet, which was honest, but I should have opened the page in step 4. Found only when the screenshot script hung waiting for the message. Fixed with one line.
7. **Screenshot tooling.** Playwright's Chromium download hung for 500+ seconds with no output; I then ran a watcher on a cache folder that was never going to fill. I only looked for an installed browser after you asked about stuck tasks. Google Chrome was already on the machine. My capture script also had a race (counted tasks after the click). Both were my errors, not the app's.
8. **Doc/code mismatch.** The design said errors are `{"error":...}`, but a wrong HTTP method returns Go's plain-text 405. Found during the final check; you chose to keep it, and the README and design doc now state it.
9. **Missed the multi-window case.** Neither the plan nor my tests considered two windows. You found it by using the app.
10. **Went quiet.** After step 4 I stopped without telling you; you had to ask "why did you stop" (prompt 8). I can't reconstruct the reason from the record, which is itself the problem.
11. **Validation was too thin until you asked.** I only checked empty/length, so control characters and invalid UTF-8 were stored. SQL injection and XSS were already prevented by design (placeholders, `textContent`), but I never said so in the plan, and cross-site form posts could add tasks. Fixed after prompt 19.
12. **Found by the independent review, not by me:** shutdown closed the DB while requests could still be draining; after editing a task deleted elsewhere the page stopped polling; typed edit text was lost when another row changed. The last two came from my own multi-window change, which I only tested on the happy path.
13. **I wrote claims I had not verified.** The README said SQLite creates `-wal`/`-shm` files (WAL was never enabled), and the design doc said every mutation is atomic (toggle is not). Corrected.
14. **Check-script hygiene.** My first run of the new browser script hung because the failed script left its server running and holding the pipe; I also mistyped a loop's arguments once. Test-script issues, not app bugs.
15. **Prompt log gaps.** The rule was "append every message at the start of each turn". I let prompts 10-12 and 13-15 pile up and caught up later.

Nothing else broke: handler tests passed on first run, and all suites are green uncached with `-race`.

## (c) What could have been better

**You**
- Scale intent came after the design (prompt 3). Saying "may become multi-user" up front would have skipped the JSON option. It cost only a doc rewrite since no code existed, but it was avoidable.
- Acceptance criteria didn't mention behaviour across windows or live updates; you found the gap manually. One line ("works with the app open in two tabs") would have covered it.
- The environment (old Go, no golangci-lint) wasn't mentioned, and "lint and tests pass" was a hard criterion. You resolved it quickly in prompt 2.
- Security and "who else can reach this" were not in the requirements; they came as a follow-up after the build. One line ("treat all input as hostile; localhost only") up front would have put the checks in the plan.
- Prompt 1 was otherwise strong: clear scope, explicit out-of-scope list, ordered execution rules, honest-screenshots rule.

**Me**
- Check the environment (Go, git, linter versions) before writing the plan, not mid-step.
- For anything with a UI, open it in a browser before calling the step done; curl proves the API, not the page.
- Tell you when I stop, and say why.
- Verify download tooling is making progress after a minute, not after ten.
- Keep `prompt.md` current each turn as instructed.
- Put input validation and a security pass in the plan, and run an independent review before saying a feature is done, not after.
- Test my own new features beyond the happy path (stale windows, deleted records) when I add them.
- Verify README/doc claims against the code before writing them.

## (d) Suggestions for the workflow and CLAUDE.md

1. **Define the no-remote case.** Section 8/9 says push after each step; prompt 1 said skip the push if no remote. Put "no remote: commit locally, don't ask" in CLAUDE.md.
2. **Language.** CLAUDE.md says chat in Chinese, but you wrote English throughout and I followed that. State the rule you want.
3. **UI steps need a browser check.** Add to section 9.2: for steps that touch a frontend, "done" requires running it in a browser, not only unit tests.
4. **Environment preflight.** Add a first-step checklist: Go version, linter present, git version. Today those surprises cost several rework items in step 1.
5. **Scale intent in section 10.** Make "expected users/instances" a required line under Requirements, so persistence choices aren't revisited mid-plan.
6. **Keep the "Mistakes / rework" line** in the dev-log template permanently; it made this reflection quick and checkable.
7. **Add a review step.** A fresh-context subagent (no hints) found real issues that my own checks missed; make it a standard last step before commit/push.
8. **Add a security line to section 10's Requirements** (input rules, bind address, cross-site posts) so it is decided up front.
9. **Decide whether the browser script belongs in the repo.** It found the real bug, but lives in `/tmp` and isn't run by `make test`. Either commit it as an optional `make e2e` (adds a Node dev dependency) or accept manual checks.
