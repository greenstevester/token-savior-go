# M1 Go Port — Execution Status

> Companion to `2026-05-14-go-port-m1.md`. Captures progress + carry-forwards as the plan is executed across multiple sessions.

**Branch:** `feat/go-port-m1`
**Last completed:** **T24** (Update top-level docs + close compat parity) — **final checkpoint reached, M1 complete**
**Last commit:** `873e7ca` `docs: announce Go port M1 in README and CLAUDE.md` (T24-F status update follows)
**Binary builds and boots:** `WORKSPACE_ROOTS=$(pwd) ./bin/token-savior < /dev/null` exits on EOF after emitting `[token-savior] profile= version=… commit=… roots=1`
**Compat harness:** `./bin/ts-compat -fixture testdata/fixtures/go-small -python ~/.venvs/token-savior/bin/token-savior` exits 0 with `All tools matched` (1 SKIP for `list_workspace_roots`, 5 OK).

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
| T21 | Compat harness | ✅ | `1f20d55` (live diffs surfaced 2 compat-deltas, see notes #15–#16) |
| T22 | Baseline capture + manifest sizing | ✅ | `b5ed12b` (manifest: Go 1815 B vs Python 33272 B for full profile — gate met 18×) |
| T23 | GitHub Actions CI (Go) | ✅ | `95d32ad` — **checkpoint reached** (compat-harness non-blocking until T24) |
| T24 | Update top-level docs + close compat parity | ✅ | `873e7ca` (5 commits: harness wiring `8a74504` + expected_diffs.go `918edb4` + CI revert `153050a` + docs `873e7ca` + status update) — **final checkpoint, M1 complete** |

**User-requested checkpoints (stop for review):** T1 ✅, T7 ✅, T11, T20, T21, T23, T24.

## What landed at T24

- **Harness clean-JSON wiring (`8a74504`):** `cmd/ts-compat/main.go` now sets `TS_NO_HINTS=1` in the subprocess env and passes `"compress": false` in the args of every compressible M1 tool. Sidesteps Python's `@F:/@S:/@T:/@P:` compact DSL by asking Python to skip compression — no parser needed. Carry-forward #15 resolved.
- **Expected-diffs whitelist (`918edb4`):** New `internal/compat/expected_diffs.go` (~253 LoC) with per-tool `Expected{Skip, Reason, Normalize}` entries for the 5 M1 query tools (find_symbol, get_functions, get_classes, get_imports, search_codebase). Normalizers translate Python's wire shape into Go's — generic helpers `dropKeys`, `renameKeys`, `linesToLineEndline`, `wrapDictAsArray` make each per-tool function 3-5 calls. `list_workspace_roots` is `Skip: true` (Python v3 doesn't implement it). Comprehensive table-driven tests in `internal/compat/expected_diffs_test.go` (21 tests, all green; both happy + negative path per tool plus helper unit tests). Carry-forwards #16 (`search_codebase` field renames) and #1 (`Function.Signature`) resolved as part of this.
- **CI workaround revert (`153050a`):** `.github/workflows/ci-go.yml` drops `continue-on-error: true` from the compat-harness job and re-gates `build` on `[test, lint, compat-harness]`.
- **Docs (`873e7ca`):** README banner inserted after line 22 announcing the v4 Go port. CLAUDE.md gets a `## Go port (in progress, M1)` section after Orientation, plus the Orientation paragraph itself is corrected (used to say "There is no Go code here; the suffix is historical" — now reflects the v3+v4 split). The latter is **plan-bug #18** (plan didn't anticipate that the Orientation contradiction needed fixing alongside the new section).
- **Status doc (this commit):** task table marked complete, carry-forwards #15/#16/#1 moved to Resolved, #3 (annotator edge-case tests) flagged as the only remaining carry-forward for the M1 PR description.

### Compat parity decision (T24 architectural call)

The naive read of carry-forwards #15/#16/#1 was "rename a few Go fields and we're done." The reality discovered when the harness produced clean diffs was that **every M1 list-returning tool diverges structurally** (Python has `{name, qualified_name, lines:[s,e], params, is_method, parent_class, file}` for functions; Go has `{file, qualified, line, end_line, signature}`). Per spec §337/§374, E1 allows documenting divergences in `internal/compat/expected_diffs.go` as an explicit escape hatch, so we took the **whitelist-first** posture: keep Go's idiomatic field names (`qualified`, `line/end_line`, `kind`, `signature`) and translate Python's response into Go's shape before diffing. Rationale: (a) E1's "shape compare" provision sanctions this, (b) Go's internal field set is genuinely richer in places (e.g., `ClassHit.Kind` distinguishing struct/interface/alias has no Python analogue), (c) cheaper than a multi-file Go refactor for a port that's still pre-cutover. Alternative options considered: rename Go to match Python verbatim (rejected — substantial Go churn, loses info), hybrid (rejected — only `search_codebase` was an obvious-rename case; the others are more structural). One fidelity-gap note carried in the `find_symbol` normalizer's `Reason`: Python's cross-language Go annotator emits Python-style signatures (`"def Hello()"`) for Go code, so `signature` is dropped from both sides during comparison — this hides Python's wrong-signature output rather than the data-identity fields (`file`/`line`/`qualified`/`kind`), which still diff if they disagree.

## What landed at T22-T23

- `cmd/ts-cli manifest` (build via `make build-ts-cli`) — per-profile byte counts (`tool_count`, `name_bytes`, `desc_bytes`, `schema_bytes`, `total_bytes`). M1 baseline: every profile shows 8 tools / 1815 bytes total (all tagged `AllProfiles`).
- `scripts/capture-baselines.sh` — records Python v3 cold-boot latency + harness parity status to `testdata/baselines/python-v3-<date>.json`. Honours `TS_PYTHON_BIN` env var (defaults to `token-savior` on PATH). Captured 2026-05-16: `cold_index_ms=270`, `tools_parity=false`.
- `.github/workflows/ci-go.yml` — five jobs: `test` (race + coverage), `lint` (golangci-lint), `compat-harness` (non-blocking until T24), `security` (gosec, no-fail), `build` (linux amd64/arm64 static binaries as artefacts). `build` gates on `test+lint` only (not compat-harness). Existing Python `ci.yml` untouched.

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

## Carry-forward — resolved at T24

- **#1 `Function.Signature` vs Python's `signature_hash`** — RESOLVED in `918edb4`. The `find_symbol` normalizer drops `signature` from both sides (Python's cross-language annotator emits Python-style signatures for Go code — fidelity gap rather than identity divergence). `get_functions` normalizer drops Go's `signature` because Python doesn't emit it in that output at all.
- **#15 Python compact-text format** — RESOLVED in `8a74504`. Harness passes `"compress": false` in args + sets `TS_NO_HINTS=1` in env; Python returns raw JSON.
- **#16 `search_codebase` field rename** — RESOLVED in `918edb4`. Whitelist-first via `expected_diffs.go` normalizer (`content`→`text`, `line_number`→`line` on Python side). Go's `{file, line, text}` field names preserved as the wire shape.

## Carry-forward (deferred polish; not blocking M1)

1. **`Function.Signature` vs Python's `signature_hash`** — ~~structural mismatch the T21 compat harness must resolve. Likely adds a `SignatureHash` field to Go's `Function` and either remaps or ignores the text `Signature` in the diff. **Will surface at T21.**~~ **Resolved at T24** — see above.
2. **`renderFuncSignature` error fallback** returns bare name without `func ` prefix — unreachable today.
3. **Annotator edge-case tests** missing: empty file, parse error, types-only file. *Did not land before T9.* The indexer happily collects per-file annotation errors via `errors.Join`, but the parse-error path isn't covered by a test yet — slot in before T12 if convenient, otherwise during T20 handler tests. **Still open at M1 close** — flag in the M1 PR description as the only deferred carry-forward; low risk to ship without (path is hit + observed by the cache-rebuild flow already), but worth landing early in M2.
4. **Uppercase-extension test** for `LanguageForPath` (`FOO.GO` → `"go"`) — proves the `strings.ToLower` branch.
5. **Module path** is bare `token-savior-go` not `github.com/...`. Rename only if it becomes importable.
6. **`unused` lint exclusion on `_test.go`** is broad — tighten as the test suite grows.
7. **`make build-linux` doesn't cover `ts-compat`** — add by T23 if CI needs it.
8. **`VERSION=v1.2.3 make build` env override** no longer works after the `:=` change in T1. Restore via `?=` + a separate `BUILD_TIME := …` if a release flow needs it (probably alongside T23 goreleaser setup).
9. **Plan-bug audit** — two were caught during T1–T7 (T3 missing `.jsx`, T4 missing generic-receiver cases). T9 had a "broken-then-corrected" `init()` block in the plan text that the controller collapsed in the dispatch prompt (kept only the corrected version, dropped the `imports()` stub). Worth continuing the audit before each subsequent dispatch. **T24** caught plan-bug #18: the plan body said to insert a `## Go port` section after `## Orientation` in CLAUDE.md, but didn't mention that the existing Orientation paragraph stated "There is no Go code here; the suffix is historical" — which becomes a self-contradiction the moment the new section lands. Fixed inline at T24-E by replacing the contradictory bullet with a v3+v4 split description.
10. **gosec 0o644 noise** — the plan uses `0o644` for test-file writes; gosec G306 flags it. Pattern across T8/T9/T11 is to tighten to `0o600` (or `0o750` for dirs) in the dispatch prompt. Pre-existing test fixtures still use `0o644` — leave them alone unless we touch the file for another reason.
11. **`bytes` shadow noise** — plan test code repeatedly uses `bytes, _ := json.Marshal(...)` as a local var, which shadows the stdlib `bytes` package. Always rename to `raw` (or similar) in dispatch prompts. Hit at T17 and T20.
12. **mark3labs API drift** — plan referenced `mcp.WithInputSchemaRaw` but the v0.54.0 export is `mcp.WithRawInputSchema`. T19 already adapted. Worth pre-checking other library symbols (`NewToolResultError`, `NewToolResultText`, `CallToolRequest.Params.Arguments`) if they appear in future tasks; all three were valid in v0.54.0 at T19 time.
13. **`make build-token-savior` target name** — plan mentions this in T20 step 6 but the Makefile target is just `make build`. Binary lands at `./bin/token-savior`. Plan's command was wrong; T20 implementer adapted. Worth verifying any plan-doc `make` invocations against the actual Makefile.
14. **Plan's silent-discard unmarshal pattern** for optional-args handlers (`_ = json.Unmarshal(raw, &args)`) trips `errcheck` lint. T20 wrapped it in `if len(raw) > 0 { if err := …; err != nil { return nil, err } }`. Net effect: malformed JSON now errors rather than being silently treated as missing — strictly correct.
15. **T21 compat-delta — Python compact-text format.** ~~The harness assumes both servers emit raw JSON in the MCP text-content field. Reality: Python's `token-savior` returns a token-saving compact format that starts with `@` (or `E` for the empty-result marker).~~ **Resolved at T24-A (`8a74504`)** — neither (a), (b), nor (c) from the original options: instead, harness now passes `"compress": false` in args (Python's `_maybe_compress` honours `arguments.get("compress", True)`) plus sets `TS_NO_HINTS=1` in env. No parser written, no Python knob added, no whitelist for this specific issue — the cleanest path was to ask Python to skip compression entirely.
16. **T21 compat-delta — `search_codebase` field rename.** ~~Python emits `{content, file, line_number}` per hit; Go emits `{file, line, text}`. Same data, different field names. **Decision needed:** rename Go fields ... or whitelist as a tolerated rename in `expected_diffs.go`.~~ **Resolved at T24-B (`918edb4`)** — whitelist-first via `expected_diffs.go` normalizer (`content`→`text`, `line_number`→`line` translated on Python side). Go's field names preserved as the canonical wire shape. Same posture applied to the deeper get_functions/get_classes/get_imports/find_symbol structural divergences also discovered at T24 (see "Compat parity decision" above).
17. **Python venv setup for the harness.** `/opt/homebrew/bin/python3 -m venv ~/.venvs/token-savior && ~/.venvs/token-savior/bin/pip install -e ".[mcp]"` produces a working v3 install. Run the harness with `./bin/ts-compat -fixture "$(pwd)/testdata/fixtures/go-small" -python ~/.venvs/token-savior/bin/token-savior`. CI will need its own pip-install step (T23).

## Operational notes (lessons from T1–T7)

- **Model:** sonnet is fine for every task type so far (config, mechanical Go, AST work). No need to escalate to opus.
- **Skip formal re-review on small/clean diffs.** Used local `git show` inspection on T5/T6/T7 instead of dispatching a quality-reviewer subagent. Saved ~5 dispatches. Risk: missing regressions. Mitigation: any diff > 80 LoC, or any task touching shared infra (`go.mod`, Makefile, CI), always gets the full two-stage review.
- **SendMessage to resume** the same implementer subagent is the right pattern when feeding fix-list reviews back. Avoids re-paying context cost.
- **Subagents pause asking for commit approval** even though commits are pre-authorised — telling them "go ahead" via SendMessage is one round-trip per task. Could try setting `--dangerously-skip-permissions` on the subagent dispatch, but that's a bigger lever.
- **Three-commit chains per task** are normal once spec + code reviews surface issues. Aim to fold fixes into the original commit only when the work is genuinely amenable — usually it isn't, and the per-commit history aids bisect.
- **`.claude/settings.json` allowlist landed mid-stream** to reduce prompt noise. Contains `defaultMode: "acceptEdits"` + ~45 Bash entries.

## How to resume

M1 is complete. The next step is Steve opening the M1 PR (`feat/go-port-m1` → `main`) — handled out of session per the T24 resume prompt. After that, `sg-document-release` for the post-merge doc sweep.

For M2 planning, the carry-forwards still open are:
- #3 (annotator edge-case tests) — flag in PR description; ship early in M2.
- The `find_symbol` `signature` fidelity gap noted in `expected_diffs.go`'s Reason string — only relevant when M2 adds language annotators that the Python side already covers (Java, TS, Rust, shell). Until then, the normalizer is correctly dropping noise.

Historical note for future plan executors: tasks contain complete TDD-shaped steps with verbatim Go code, test fixtures, expected commands, and expected outputs. Use the `sp-subagent-driven-development` skill to dispatch tasks one at a time. The fix-loop pattern (implementer → spec reviewer → fix → code reviewer → fix → done) worked well across T1–T24; lean on it for any task that touches shared infrastructure or has non-trivial logic. Smaller, mechanical tasks (T5–T7 style) can skip the formal code-quality re-review once the diff is visibly clean — but document why if you do.
