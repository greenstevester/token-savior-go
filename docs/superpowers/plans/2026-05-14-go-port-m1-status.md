# M1 Go Port — Execution Status

> Companion to `2026-05-14-go-port-m1.md`. Captures progress + carry-forwards as the plan is executed across multiple sessions.

**Branch:** `feat/go-port-m1`
**Last completed:** **T11** (Slot + Manager + ParseWorkspaceRoots) — **first checkpoint**
**Last commit:** `ec6eb02` `feat(slot): per-project slot lifecycle and manager`
**Total commits on branch:** 19 (1 plan + 1 status doc + 17 implementation)

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
| T11 | Slot + SlotManager + ParseWorkspaceRoots | ✅ | `ec6eb02` — **checkpoint reached** |
| T12 | Query — FindSymbol | ⏳ pending | |
| T13 | Query — GetFunctions / GetClasses / GetImports | ⏳ pending | |
| T14 | Query — SearchCodebase | ⏳ pending | |
| T15 | Tool registry + ProfileSet + 8 M1 schemas | ⏳ pending | |
| T16 | Profile parsing + visibility filter | ⏳ pending | |
| T17 | Session stats counters | ⏳ pending | |
| T18 | MCP ToolContext + Dispatcher | ⏳ pending | |
| T19 | MCP stdio server + `cmd/token-savior/main.go` | ⏳ pending | |
| T20 | M1 tool handlers + SlotView adapter | ⏳ pending — **checkpoint** | |
| T21 | Compat harness | ⏳ pending — **checkpoint** | |
| T22 | Baseline capture + manifest sizing | ⏳ pending | |
| T23 | GitHub Actions CI (Go) | ⏳ pending — **checkpoint** | |
| T24 | Update README + CLAUDE.md | ⏳ pending — **final checkpoint** | |

**User-requested checkpoints (stop for review):** T1 ✅, T7 ✅, T11, T20, T21, T23, T24.

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
