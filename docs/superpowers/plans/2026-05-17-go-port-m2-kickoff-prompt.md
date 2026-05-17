Begin executing the M2 plan for the token-savior Go port. Project root:
/Users/stevengreensill/dev/git-repos/github/ai/token-savior-go

Create a new branch: feat/go-port-m2 (from main, currently at dba389a).

Read these in order — they're the full context:
  - docs/superpowers/plans/2026-05-17-go-port-m2.md  (41-task plan, ~11k
    lines; the authoritative spec for what to build)
  - docs/superpowers/specs/2026-05-14-port-to-go-design.md  (the master
    design spec; §169 for "hand-rolled tokenizers, NO tree-sitter, NO
    WASM", §467 for the M2 row, §336–§341 for exit gates E1/E2/E5)
  - docs/superpowers/plans/2026-05-14-go-port-m1.md  +
    docs/superpowers/plans/2026-05-14-go-port-m1-status.md  (M1 execution
    history — the operational patterns, model choices, and 17 plan-bugs
    that surfaced during M1. M2 needs the same vigilance.)

State of the repo:
  - M1 shipped (PR #1) — 8 MCP tools, Go annotator, compat harness.
  - chore/post-m1-followup PR #2 merged — .gitignore for stray binaries,
    annotator edge-case tests (empty/parse-error/types-only), status doc
    sync. All M1 carry-forwards now closed.
  - The M2 plan was authored by a subagent and committed via PR #3.
    Steve has not yet reviewed the plan in depth — that's task zero of
    this session before any code lands.

The M2 plan has 41 tasks across 7 phases:
  Phase 1 (T1-T4):   Shell annotator           (smallest, proves the
                                                non-Go annotator path)
  Phase 2 (T5-T12):  TypeScript annotator      (largest — JSX/TSX/template
                                                literals)
  Phase 3 (T13-T18): Rust annotator
  Phase 4 (T19-T24): Java annotator
  Phase 5 (T25-T33): 7 nav tools + schemas/handlers + expected_diffs.go
                     extensions
  Phase 6 (T34-T38): Exit-gate tooling (E5 fidelity, E2 cold-index,
                     per-language fixtures, CI matrix, baseline capture)
  Phase 7 (T39-T41): Docs + status doc

Sequencing locked-in: shell → TS → Rust → Java → nav tools → exit-gate
tooling → docs. Ship cadence: ONE M2 PR (feat/go-port-m2 → main),
matching M1's single-PR shape.

User-requested checkpoints — STOP for Steve's review at:
  - T4  (shell annotator complete; non-Go path proven)
  - T12 (TS annotator complete; biggest scanner work landed)
  - T18 (Rust complete)
  - T24 (Java complete; all 4 annotators live)
  - T33 (all 7 nav tools + compat normalisers landed)
  - T38 (E1/E2/E5 baselines captured; gate readings exist)
  - T41 (M2 PR ready to open — final checkpoint)

NINE in-plan judgment calls the plan author flagged for revisit during
execution. Read each task's body for the inline rationale before
implementing — these are deliberate scope cuts, not oversights:
  1. Shell aliases → recorded as Function records (no Alias kind in
     models). T1-T4.
  2. Rust `mod foo {…}` → recorded as Class{Kind:"alias"}. T15.
     Consider whether a Kind="module" addition belongs in M3+.
  3. Java `record` → Class{Kind:"class"}. T22. Minor parity gap.
  4. Java annotations parsed-but-discarded. T23. Function.Annotations
     []string is M3+ work.
  5. Java call edges chain only 2 idents — `this.owners.findById`
     collapses to `this.owners`. T24's note: implementer chooses
     between extending parseCalls to N-ident chains OR adding an
     expected_diffs.go normaliser.
  6. Shell alias names follow POSIX regex (`name`); bash allows
     `deploy-prod`. T4. Widen if T34 fidelity gate flags.
  7. E2 CI hook is soft — T35 writes JSON with gate_status, T37 doesn't
     `jq -e` on it. Flagged in self-review as carry-forward.
  8. get_call_chain ignores Python's `level` parameter. T30. Hop-
     verbosity knob deferred to M5+.
  9. Shell fidelity is structural self-check (Python has no shell
     annotator per spec §175). T34's script writes recall=precision=1.0
     for shell and skips the diff; auto-becomes real comparison if
     Python ever adds shell.

Tooling state (unchanged from M1):
  - go 1.26.2, golangci-lint 2.12.1
  - mark3labs/mcp-go v0.54.0 (note: it's WithRawInputSchema, not
    WithInputSchemaRaw — M1 plan-bug #12)
  - .claude/settings.json allowlist + defaultMode: "acceptEdits" set up
  - Makefile targets: build, build-ts-compat, build-ts-cli, build-all,
    build-linux, clean, test, test-compat, lint
  - Python v3 venv for the harness: ~/.venvs/token-savior, drives
    ./bin/ts-compat with -python ~/.venvs/token-savior/bin/token-savior

Use the sp-subagent-driven-development skill. Pattern from M1 (worked
across all 24 tasks):
  - sonnet model for implementer + reviewers; no need for opus
  - dispatch implementer with the full task text verbatim from the plan
  - small clean diffs (skeleton/scanner tasks; mechanical refactors)
    skip formal code-quality re-dispatch once the diff is visibly clean
  - shared-infra tasks (tools/schemas.go, mcp/handlers.go,
    compat/expected_diffs.go, Makefile, CI) ALWAYS get the full two-
    stage spec+code review
  - commits per task are pre-authorised on feat/go-port-m2; if a
    subagent pauses for permission, "go ahead"
  - SendMessage to resume the same implementer when feeding fix-list
    reviews back

M2 plan-bug audit: M1 surfaced 18 plan-bugs across 24 tasks. M2 has
~2.5x the surface area. Stay vigilant — verify each referenced symbol
exists in the codebase before dispatching, and audit the plan's stated
behaviour against current reality. Common M1 traps (status doc §9):
  - bytes-var-shadow (use `raw` not `bytes` for local JSON vars)
  - 0o644 on test-file writes (use 0o600 — gosec G306)
  - silent-discard `_ = json.Unmarshal(...)` for optional args
    (errcheck — use the guarded pattern from M1's T20)
  - referencing make targets that don't exist (verify against Makefile)

T41 is the final checkpoint — STOP for Steve's review. Don't auto-push
or auto-open the M2 PR; Steve handles that.

Begin by reading the three plan/status docs in full, then propose the
order you'll work through (likely: read-plan → T1-T4 [shell, dispatch
to subagent] → checkpoint → T5-T12 [TS] → checkpoint → etc.). Ask once
if anything in the plan or judgment calls is ambiguous, then execute.
