# M1 Go Port — Execution Status

> Companion to `2026-05-14-go-port-m1.md`. Captures progress + carry-forwards as the plan is executed across multiple sessions.

**Branch:** `feat/go-port-m1`
**Last completed:** **T21** (Compat harness — built, dry-run produced real diffs)
**Last commit:** `1f20d55` `feat(compat): add Python v3 / Go v4 diff harness`
**Binary builds and boots:** `WORKSPACE_ROOTS=$(pwd) ./bin/token-savior < /dev/null` exits on EOF after emitting `[token-savior] profile= version=… commit=… roots=1`

## Task progress (24 total)

| # | Task | Status | Final SHA |
|---|---|---|---|
| T1 | Bootstrap Go module | ✅ | `141b6e2` (3 commits) |
| T2 | Core data models | ✅ | `9387a5a` (2 commits) |
| T3 | Annotator interface + dispatch + exclusions | ✅ | `bf973f3` (2 commits) |
| T4 | Go annotator — functions and methods | ✅ | `62f8d2c` (3 commits: incl. doc-comment fix + generic receivers) |
| T5 | Go annotator — types | ✅ | `d65bea0` |
| T6 | Go annotator — imports | ✅ | `6f6c15e` |
| T7 | Go annotator — call edges | ✅ | `90d5e56` |
| T8 | Project walker | ✅ | `b15cc84` |
| T9 | ProjectIndexer with worker pool | ✅ | `61ae3c6` |
| T10 | Symbol-body hashing | ✅ | `95c92b2` |
| T11 | Slot + SlotManager + ParseWorkspaceRoots | ✅ | `ec6eb02` |
| T12 | Query — FindSymbol | ✅ | `bc646e3` |
| T13 | Query — GetFunctions / GetClasses / GetImports | ✅ | `1846003` |
| T14 | Query — SearchCodebase | ✅ | `156110b` |
| T15 | Tool registry + ProfileSet + 8 M1 schemas | ✅ | `2cc783f` |
| T16 | Profile parsing + visibility filter | ✅ | `d7d09a1` |
| T17 | Session stats counters | ✅ | `58833b2` |
| T18 | MCP ToolContext + Dispatcher | ✅ | `ea0d53d` |
| T19 | MCP stdio server + `cmd/token-savior/main.go` | ✅ | `b157d59` |
| T20 | M1 tool handlers + SlotView adapter | ✅ | `14d6ad0` |
| T21 | Compat harness | ✅ | `1f20d55` — **checkpoint reached** (live diffs surfaced 2 compat-deltas, see notes #15–#16) |
| T21 | Compat harness | ⏳ pending — **checkpoint** | |
| T22 | Baseline capture + manifest sizing | ⏳ pending | |
| T23 | GitHub Actions CI (Go) | ⏳ pending — **checkpoint** | |
| T24 | Update README + CLAUDE.md | ⏳ pending — **final checkpoint** | |

**User-requested checkpoints (stop for review):** T1 ✅, T7 ✅, T11, T20, T21, T23, T24.

## What the M1 server does end-to-end (after T20)

- `cmd/token-savior` parses `WORKSPACE_ROOTS` (comma-separated) or legacy `PROJECT_ROOT`, indexes each root via `slot.Manager.RegisterRoot`, builds a `mcp.ToolContext{SlotManager, Stats}`, wires 8 handlers via `mcp.RegisterHandlers(dispatcher)`, advertises via `mcp.Serve` filtered by `tools.ParseProfile(os.Getenv("TOKEN_SAVIOR_PROFILE"))`, then runs `server.ServeStdio`.
- Tool handlers (`internal/mcp/handlers.go`): `find_symbol`, `get_functions`, `get_classes`, `get_imports`, `search_codebase`, `switch_project`, `list_workspace_roots`, `get_stats`. Each delegates to `internal/query`; `query.SlotView{Root, Index}` is the adapter that keeps `internal/mcp` from importing `internal/slot`'s concrete type.
- Optional-args handlers (`get_functions/classes/imports`) use a `len(raw)>0` guard then propagate unmarshal errors — strictly correct vs. the plan's silent-discard pattern (errcheck lint required the change).
- Stats: every dispatch increments `ToolCalls[name]` and `TotalChars` regardless of handler outcome. `get_stats` returns the snapshot.
- mark3labs/mcp-go API note: plan referenced `mcp.WithInputSchemaRaw`; actual export at v0.54.0 is `mcp.WithRawInputSchema(json.RawMessage)`.

## What the indexer/slot layer does (after T11)

- `indexer.Walk(root)` — forward-slash relative paths, annotator-aware filtering (skips dirs hit by `IsPathExcludedFromScans`, skips files without a registered language).
- Annotators self-register via `annotator.Register("go", New())` in their package `init()`. Consumers blank-import the language sub-package (`_ "token-savior-go/internal/annotator/golang"`) to wire it in.
- `indexer.NewProjectIndexer(root).Build()` — NumCPU worker pool, errors joined via `errors.Join`, returns the index even on partial failure. Populates `SymbolTable` (first-seen wins), `DepGraph`, `ImportGraph`, `BasenameMap`, `SortedPaths`.
- `indexer.SymbolHash(body)` — first 8 bytes of SHA-256 hex-encoded (16 chars). Standalone, no callers yet.
- `slot.Slot` — `Root`, `Index`, `atomic.Uint64` CacheGen, `sync.RWMutex`. `BumpCacheGen` after any mutation.
- `slot.Manager` — `RegisterRoot` indexes on register, first root becomes active. `Get`/`Active`/`Switch` are RW-mutex-guarded. `Switch` is idempotent. `ParseWorkspaceRoots` splits/trims the comma-separated env var.

## What the Go annotator handles (after T7)

- Top-level funcs and methods (value + pointer receivers)
- Generic receivers (`*ast.IndexExpr`, `*ast.IndexListExpr`): `func (c *Container[T]) Get()`
- Types: struct, interface, alias (defined types grouped under `"alias"`)
- Imports: grouped + ungrouped blocks, renamed aliases, blank/dot tolerated
- Call edges: bare idents, `pkg.Ident`, `recv.Method`; non-ident receivers skipped (lossy by design)
- Path exclusion: `.git`, `node_modules`, `__pycache__`, `.token-savior-checkpoints` (vendor NOT excluded — matches Python)
- Extension dispatch for 5 langs (Go is the only one with a real annotator; Java/TS/Rust/shell are placeholders for M2)

## Carry-forward (deferred polish; not blocking M1)

1. **`Function.Signature` vs Python's `signature_hash`** — structural mismatch the T21 compat harness must resolve. Likely adds a `SignatureHash` field to Go's `Function` and either remaps or ignores the text `Signature` in the diff. **Will surface at T21.**
2. **`renderFuncSignature` error fallback** returns bare name without `func ` prefix — unreachable today.
3. **Annotator edge-case tests** missing: empty file, parse error, types-only file. *Did not land before T9.* The indexer happily collects per-file annotation errors via `errors.Join`, but the parse-error path isn't covered by a test yet — slot in before T12 if convenient, otherwise during T20 handler tests.
4. **Uppercase-extension test** for `LanguageForPath` (`FOO.GO` → `"go"`) — proves the `strings.ToLower` branch.
5. **Module path** is bare `token-savior-go` not `github.com/...`. Rename only if it becomes importable.
6. **`unused` lint exclusion on `_test.go`** is broad — tighten as the test suite grows.
7. **`make build-linux` doesn't cover `ts-compat`** — add by T23 if CI needs it.
8. **`VERSION=v1.2.3 make build` env override** no longer works after the `:=` change in T1. Restore via `?=` + a separate `BUILD_TIME := …` if a release flow needs it (probably alongside T23 goreleaser setup).
9. **Plan-bug audit** — two were caught during T1–T7 (T3 missing `.jsx`, T4 missing generic-receiver cases). T9 had a "broken-then-corrected" `init()` block in the plan text that the controller collapsed in the dispatch prompt (kept only the corrected version, dropped the `imports()` stub). Worth continuing the audit before each subsequent dispatch.
10. **gosec 0o644 noise** — the plan uses `0o644` for test-file writes; gosec G306 flags it. Pattern across T8/T9/T11 is to tighten to `0o600` (or `0o750` for dirs) in the dispatch prompt. Pre-existing test fixtures still use `0o644` — leave them alone unless we touch the file for another reason.
11. **`bytes` shadow noise** — plan test code repeatedly uses `bytes, _ := json.Marshal(...)` as a local var, which shadows the stdlib `bytes` package. Always rename to `raw` (or similar) in dispatch prompts. Hit at T17 and T20.
12. **mark3labs API drift** — plan referenced `mcp.WithInputSchemaRaw` but the v0.54.0 export is `mcp.WithRawInputSchema`. T19 already adapted. Worth pre-checking other library symbols (`NewToolResultError`, `NewToolResultText`, `CallToolRequest.Params.Arguments`) if they appear in future tasks; all three were valid in v0.54.0 at T19 time.
13. **`make build-token-savior` target name** — plan mentions this in T20 step 6 but the Makefile target is just `make build`. Binary lands at `./bin/token-savior`. Plan's command was wrong; T20 implementer adapted. Worth verifying any plan-doc `make` invocations against the actual Makefile.
14. **Plan's silent-discard unmarshal pattern** for optional-args handlers (`_ = json.Unmarshal(raw, &args)`) trips `errcheck` lint. T20 wrapped it in `if len(raw) > 0 { if err := …; err != nil { return nil, err } }`. Net effect: malformed JSON now errors rather than being silently treated as missing — strictly correct.
15. **T21 compat-delta — Python compact-text format.** The harness assumes both servers emit raw JSON in the MCP text-content field. Reality: Python's `token-savior` returns a token-saving compact format that starts with `@` (or `E` for the empty-result marker). 5 of 6 M1 tools fail `DiffJSON` on `"unmarshal want: invalid character '@' looking for beginning of value"`. **For T24** the options are: (a) teach `ts-compat` to decode Python's compact format before diffing, (b) add a `TS_RAW_JSON=1` server knob to v3 (violates "don't touch Python"), or (c) ship `internal/compat/expected_diffs.go` with these mismatches whitelisted. Lean: (a) — the format reader is mechanical and the harness already owns the comparison shape. Reproducer: `~/.venvs/token-savior/bin/token-savior` against `testdata/fixtures/go-small/`.
16. **T21 compat-delta — `search_codebase` field rename.** Python emits `{content, file, line_number}` per hit; Go emits `{file, line, text}`. Same data, different field names. **Decision needed:** rename Go fields to `{content, file, line_number}` to match Python (1-line change in `internal/query/search.go::SearchHit`), or whitelist as a tolerated rename in `expected_diffs.go`. The Go names are arguably better (`line` is consistent with the rest of the query result types, `text` is shorter than `content`), so the call is whether to optimise for Python-parity or for Go-internal consistency. Same answer probably applies to other field-rename diffs that'll surface once #15 is unblocked.
17. **Python venv setup for the harness.** `/opt/homebrew/bin/python3 -m venv ~/.venvs/token-savior && ~/.venvs/token-savior/bin/pip install -e ".[mcp]"` produces a working v3 install. Run the harness with `./bin/ts-compat -fixture "$(pwd)/testdata/fixtures/go-small" -python ~/.venvs/token-savior/bin/token-savior`. CI will need its own pip-install step (T23).

## Operational notes (lessons from T1–T7)

- **Model:** sonnet is fine for every task type so far (config, mechanical Go, AST work). No need to escalate to opus.
- **Skip formal re-review on small/clean diffs.** Used local `git show` inspection on T5/T6/T7 instead of dispatching a quality-reviewer subagent. Saved ~5 dispatches. Risk: missing regressions. Mitigation: any diff > 80 LoC, or any task touching shared infra (`go.mod`, Makefile, CI), always gets the full two-stage review.
- **SendMessage to resume** the same implementer subagent is the right pattern when feeding fix-list reviews back. Avoids re-paying context cost.
- **Subagents pause asking for commit approval** even though commits are pre-authorised — telling them "go ahead" via SendMessage is one round-trip per task. Could try setting `--dangerously-skip-permissions` on the subagent dispatch, but that's a bigger lever.
- **Three-commit chains per task** are normal once spec + code reviews surface issues. Aim to fold fixes into the original commit only when the work is genuinely amenable — usually it isn't, and the per-commit history aids bisect.
- **`.claude/settings.json` allowlist landed mid-stream** to reduce prompt noise. Contains `defaultMode: "acceptEdits"` + ~45 Bash entries.

## How to resume

Read `docs/superpowers/plans/2026-05-14-go-port-m1.md` for the full task code blocks. Each task contains complete TDD-shaped steps with verbatim Go code, test fixtures, expected commands, and expected outputs.

Use the `sp-subagent-driven-development` skill to dispatch tasks one at a time. The fix-loop pattern (implementer → spec reviewer → fix → code reviewer → fix → done) has been working; lean on it for any task that touches shared infrastructure or has non-trivial logic. Smaller, mechanical tasks (T5–T7 style) can skip the formal code-quality re-review once the diff is visibly clean — but document why if you do.
