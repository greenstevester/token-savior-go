Resume executing the M1 Go-port implementation plan. Project root:
/Users/stevengreensill/dev/git-repos/github/ai/token-savior-go

Branch: feat/go-port-m1 (currently HEAD: 4588799, 33 commits ahead of main).

Read these two files first — they're the full context:
  - docs/superpowers/plans/2026-05-14-go-port-m1.md         (24-task plan)
  - docs/superpowers/plans/2026-05-14-go-port-m1-status.md  (T1-T23 done,
    carry-forwards #1-#17, operational notes — read all of it, the bottom
    half is where the unresolved decisions live)

What's done: T1-T23, all 24 plan tasks except T24. Highlights:
  - Go MCP server boots, all 8 M1 tools wired, ./bin/token-savior works
  - Compat harness ./bin/ts-compat builds and runs against Python v3 — see
    note below on venv setup
  - Manifest sizing gate is met 18x (Go 1815B vs Python 33272B for full
    profile)
  - Go CI workflow (.github/workflows/ci-go.yml) added; compat-harness
    job is intentionally non-blocking until T24 closes parity
  - Captured Python baseline at testdata/baselines/python-v3-2026-05-16.json
    (cold_index_ms=270, tools_parity=false — parity false is expected, see
    status doc notes #15-16)

Next is T24 — the FINAL checkpoint, your sign-off task. Per the plan body,
T24 covers updating README + CLAUDE.md. But there's more carry-forward
work that the status doc tracks and that you should treat as in-scope for
T24 / land before the M1 PR opens. The non-doc decisions still open:

  - Note #15: Python compact-text format. The harness's DiffJSON expects
    raw JSON in MCP text content; Python token-savior returns a token-
    saving compact format starting with '@' or 'E'. Blocks 5/6 tools.
    Decide: teach ts-compat to decode v3's compact format, OR whitelist
    via internal/compat/expected_diffs.go. Lean: teach the harness.

  - Note #16: search_codebase fields differ. Python emits {content, file,
    line_number}; Go emits {file, line, text}. Decide: rename Go's
    SearchHit fields to match Python (1-line change in internal/query/
    search.go), or whitelist as a tolerated rename. The Go names are
    arguably better for internal consistency.

  - Carry-forward #1: Function.Signature vs Python's signature_hash. Will
    show as another field-diff once #15 is unblocked. May want a
    SignatureHash field on Function, or whitelist.

  - Once #15/#16 are resolved, REVERT the CI workaround:
      * Drop `continue-on-error: true` from the compat-harness job
      * Add `compat-harness` back to the `build` job's `needs:` list
    Both changes are commented inline in .github/workflows/ci-go.yml.

  - Carry-forward #3: Annotator edge-case tests (empty file, parse error,
    types-only file) still not landed. Low-risk to ship without; mention
    in the PR description.

Tooling state:
  - go 1.26.2, golangci-lint 2.12.1, both clean across all 10 packages
  - mark3labs/mcp-go v0.54.0 (note: plan referenced WithInputSchemaRaw;
    actual export is WithRawInputSchema, already adapted at T19)
  - .claude/settings.json allowlist + defaultMode: "acceptEdits" already
    set up; most Bash you'll need won't prompt
  - Makefile targets: build (= build-token-savior), build-ts-compat,
    build-ts-cli, build-all, build-linux, clean, test, test-compat, lint

Python venv for running the harness:
  ~/.venvs/token-savior is set up with editable install of v3
  (token-savior-recall 3.0.0 + mcp 1.27.1). Drive ts-compat with:
    ./bin/ts-compat -fixture "$(pwd)/testdata/fixtures/go-small" \
                    -python ~/.venvs/token-savior/bin/token-savior
  Re-capture baseline with:
    TS_PYTHON_BIN=~/.venvs/token-savior/bin/token-savior \
      ./scripts/capture-baselines.sh

Use the sp-subagent-driven-development skill. The pattern that's been
working through T1-T23:
  - sonnet model for implementer + reviewers (no need for opus)
  - dispatch implementer with the full task text verbatim from the plan
  - small clean diffs (T5/T6/T7/T10-style) skip formal code-quality
    re-dispatch once the diff is visibly clean — verify via local
    `git show` and note in the response
  - commits per task are pre-authorised on this feature branch; if a
    subagent pauses asking for permission, tell it "go ahead"
  - SendMessage to resume the same implementer when feeding fix-list
    reviews back avoids re-paying context cost

Watch for plan bugs while reading T24 — the audit so far has caught 17
distinct deltas between the plan text and reality (all logged in the
status doc as carry-forward notes). T24 is mostly docs but if it touches
expected_diffs.go that's where the schema-decision logic for #15/#16
lives. Don't follow the plan into a stale call shape; verify each
referenced symbol exists.

T24 is a checkpoint — STOP for my review after it lands. Don't auto-push
or auto-open a PR; that's mine. Once T24 is in I'll handle the PR and
sg-document-release sweep.

Begin by reading the two plan docs in full. Then propose the order
you'll work through (probably #15 → #16 → #1 → CI revert → README →
CLAUDE.md → status doc final update). Ask once if anything's ambiguous,
then execute.
