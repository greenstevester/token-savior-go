# Port token-savior to Go — design

**Status:** draft, awaiting approval before implementation plan.
**Date:** 2026-05-14
**Owner:** steve
**Successor doc:** implementation plan (to be produced by `writing-plans` after this is approved).

## Goal

Replace the Python `token-savior` MCP server with a pure-Go reimplementation living in this repository. The Go binary becomes the official `token-savior` distribution from v4.0 onward. Python source is deleted at cutover; v3 stays available on PyPI as a frozen artefact.

## Scope decisions (locked during brainstorming)

| Decision | Choice |
|---|---|
| Coexistence model | In-place rewrite. Python kept during the port (compat harness needs it as an oracle); deleted at v4.0 cutover. |
| Runtime | Pure Go. **No CGO.** Single static binary, cross-compilable. |
| Languages supported | **Java, TypeScript (+ JSX/TSX), Go, shell (bash/sh/zsh), Rust.** 20 other annotators are dropped. |
| Parser strategy | Hand-rolled tokenizers per language. No tree-sitter, no WASM. |
| Memory engine | Kept, but **BM25/FTS5 only — no vector search**, no `sqlite-vec`, no ONNX embeddings. |
| Optimization engines | All five ported faithfully: PPM prefetcher, TCA co-activation, Leiden communities, LinUCB bandit, session warm-start. Leiden carries a slip-to-v4.1 escape hatch documented in Risks. |
| Testing | Compatibility harness with Python v3 as oracle during the port. Frozen golden fixtures after cutover. |
| Build/CI/distribution | Mirrors `bayvets/thepetpanicbutton-triage-server`: `Makefile`, `cmd/`+`internal/` layout, `.golangci.yml` v2, `.goreleaser.yaml`, Go 1.26, `ci.yml` with test+lint+integration+security+build jobs. |
| Staging | Approach A — six vertical-slice milestones, each producing a working binary gated by the compat harness. |

### Honest tensions in the scope

Two of these decisions trade against each other and the design has to name the cost:

1. **"Full faithful port" intent + 5-language matrix.** The Python project supports 25 languages and the README's headline (97% token savings) is measured against codebases that include Python, JS, Prisma, etc. The Go port is *not* a drop-in replacement for those users. **tsbench 192/192 is unreachable** with the v4 language matrix because the bench task generator emits Python and Prisma. v4 marketing rebuilds from new benchmarks against Java/TS/Go/shell/Rust codebases.
2. **"Faithful memory engine" + no vectors.** Hybrid BM25+vector RRF is downgraded to BM25 only. The contract surface (`memory_index` → `memory_search` → `memory_get`) is preserved, but semantic recall is materially weaker. Code paths for vectors, ONNX, fastembed, `sqlite-vec`, `symbol_embeddings` are deleted (not feature-flagged).

Both costs are accepted by the scope choices above. They are surfaced here so the implementation plan doesn't carry them as silent assumptions.

## Repository layout

```
token-savior-go/
├── cmd/
│   ├── token-savior/       # MCP stdio server entry
│   ├── ts-cli/             # Index inspection, memory dump (replaces scripts/ts_cli.py)
│   └── ts-compat/          # Compat harness runner (deletes at v4.0)
├── internal/
│   ├── mcp/                # Wire: stdio transport, tool registry, dispatch
│   ├── slot/               # SlotManager, Slot, cache_gen
│   ├── annotator/
│   │   ├── annotator.go    # Protocol + extension dispatch
│   │   ├── java/
│   │   ├── typescript/
│   │   ├── golang/
│   │   ├── shell/
│   │   └── rust/
│   ├── indexer/            # ProjectIndexer, dep graph, symbol hashes
│   ├── query/              # find_symbol, get_function_source, get_dependents, ...
│   ├── edit/               # replace_symbol_source, insert_near_symbol, move_symbol
│   ├── git/                # git_ops, git_tracker, checkpoints
│   ├── watcher/            # fsnotify-based watcher (replaces watchfiles)
│   ├── memory/
│   │   ├── db/             # modernc.org/sqlite + schema (FTS5 only)
│   │   ├── observations/   # CRUD + Bayesian validity
│   │   ├── search/         # BM25 ranking
│   │   ├── decay/          # TTL + LRU scoring
│   │   ├── roi/            # access counts, auto-promotion
│   │   └── distill/        # MDL distillation, contradiction detection
│   ├── opt/
│   │   ├── ppm/
│   │   ├── tca/
│   │   ├── leiden/
│   │   ├── linucb/
│   │   └── warmstart/
│   ├── tools/              # ToolSchema registry (single source of truth)
│   ├── profile/            # TOKEN_SAVIOR_PROFILE filtering
│   ├── stats/              # session counters, persistent stats
│   └── compat/             # Compat-harness diff logic (deletes at v4.0)
├── testdata/
│   ├── fixtures/           # Project fixtures (shared with compat harness)
│   └── golden/             # Frozen Python outputs (post-cutover oracle)
├── Makefile
├── .golangci.yml
├── .goreleaser.yaml
├── .github/workflows/
│   ├── ci.yml
│   └── release.yml
└── go.mod                  # module token-savior-go, go 1.26
```

- One MCP binary; tooling and compat harness are sibling `cmd/` entries shipped in the same release.
- No `pkg/` directory. Everything lives under `internal/` unless an external import need emerges later.
- `internal/compat/` + `cmd/ts-compat/` are temporary; deleted at v4.0 alongside Python.

## MCP wire & dispatch

### SDK

`github.com/mark3labs/mcp-go`. Most active Go MCP SDK, supports tool registration, JSON Schema, and stdio transport. No CGO. Direct equivalent of Python's `mcp.server.stdio`.

### Tool registry

Single source of truth lives in `internal/tools/schemas.go`:

```go
type ToolSchema struct {
    Name        string
    Description string
    InputSchema json.RawMessage
    Handler     Handler
    Profiles    ProfileSet  // bitmask: full|core|nav|lean|ultra|tiny|tiny_plus
}

var Schemas = map[string]ToolSchema{ ... }
```

The Python `TOOL_SCHEMAS` dict + `_PROFILE_EXCLUDES` sets collapse into a single registry with a per-tool profile bitmask. Duplicate tool names fail at compile/init time (Go duplicate-key error) — Python's import-time disjoint assertion becomes obsolete.

### Handler shape — collapses Python's 4 buckets to 1

```go
type Handler func(ctx *ToolContext, args json.RawMessage) (Result, error)

type ToolContext struct {
    Server   *Server          // META handlers
    Slot     *slot.Slot       // SLOT / QFN handlers (nil otherwise)
    QueryFns *query.Fns       // QFN handlers
    Memory   *memory.Engine   // MEMORY handlers
    Stats    *stats.Counters
}
```

The Python `META_HANDLERS` / `MEMORY_HANDLERS` / `SLOT_HANDLERS` / `QFN_HANDLERS` distinction becomes a documentation-level concern in the handler implementation, not a runtime dispatch concern. The dispatcher populates the appropriate fields on `ToolContext` based on the registered tool's needs (declared via a small `Needs` enum on `ToolSchema` if explicitness pays off).

### Profile filtering

`internal/profile/profile.go` reads `TOKEN_SAVIOR_PROFILE` once at startup, computes the advertised set. Hidden tools remain dispatchable by name — matches Python semantics. `tiny_plus` and `ultra` proxy tools (`ts_search`, `ts_extended`) are registered like any other tool with a handler that has access to the registry.

### `ts_search` without vectors

Python uses Nomic 768d embeddings on tool descriptions. Go v4 uses **BM25 over tool descriptions** via `bleve` or in-process FTS5. After the lang-matrix trim and Prisma/Python-specific tool removals, the v4 catalog is ~50–55 tools (down from Python's 67). Keyword search with well-keyworded descriptions is expected to match the user-visible quality of semantic search at this scale. Telemetry on miss rate after launch decides whether to revisit.

### Stdio lifecycle

`cmd/token-savior/main.go`:

1. Parse `WORKSPACE_ROOTS` (canonical) or `PROJECT_ROOT` (legacy).
2. Register roots with `slot.Manager`.
3. Boot optional viewer goroutine if `TS_VIEWER_PORT` set.
4. Start MCP server loop on stdin/stdout.
5. Shutdown on SIGTERM/SIGINT: cancel root context → drain in-flight tool calls → flush stats → close DB.

### Stats wrapper

Python's `_count_and_wrap_result` / `_format_result` become middleware in the dispatcher: counts chars returned vs naive cost, records per-tool counters, applies optional TCS schema compression and DCP differential context protocol where applicable.

## Annotators (5 languages)

Common protocol in `internal/annotator/annotator.go`:

```go
type Annotator interface {
    Annotate(path string, source []byte) (*models.StructuralMetadata, error)
}
```

`StructuralMetadata` carries functions, classes, imports, calls, references, exports — same fields as the Python `models.StructuralMetadata` dataclass.

Dispatch by file extension in `annotator.Annotate(path, source)` — same shape as Python's `_EXTENSION_MAP`.

### Per-language implementation notes

All five annotators are hand-rolled tokenizers, no tree-sitter, no external grammars:

- **Go** (`.go`): Use the standard library — `go/parser`, `go/ast`, `go/token`. This is the only language with a real AST option in stdlib. Highest-fidelity annotator in the project.
- **TypeScript** (`.ts`, `.tsx`, `.js`, `.jsx`): Hand-rolled scanner for ES module syntax, type declarations, JSX. Track brace depth, distinguish `class`/`interface`/`type`/`function`/`const fn = () =>`/`export ...`. Match Python `typescript_annotator.py`'s output for the same inputs.
- **Java** (`.java`): Tokenizer recognising `package`, `import`, class/interface/enum/record declarations, methods, annotations (`@RestController`, `@Service`, etc. for Spring detection). Lossier than tree-sitter on nested generics and lambdas; acceptable for structural annotation.
- **Rust** (`.rs`): Tokenizer for `mod`, `use`, `fn`, `struct`, `enum`, `trait`, `impl`. Macros (`macro_rules!`) get name-only annotation; we don't expand them.
- **Shell** (`.sh`, `.bash`, `.zsh`): New annotator (no Python equivalent). Recognise function definitions (`name() { … }` and `function name { … }`), aliases, source/dot imports. Modest contribution to navigation but rounds out the matrix.

### Indexer

`internal/indexer/indexer.go` walks the project root, dispatches to annotators in a worker pool (`runtime.NumCPU()`), builds:

- `Files`: path → `StructuralMetadata`
- `SymbolTable`: qualified name → location
- `DepGraph`: caller → callee edges
- `ImportGraph`: file → imported-file edges
- `BasenameMap`, `SortedPaths`

`is_path_excluded_from_scans` ported as-is — same `EXCLUDED_DIRS` (`.git`, `__pycache__`, `node_modules`, `.token-savior-checkpoints`).

Symbol-hash population (`fill_hashes` in Python) computes a SHA-256 over each symbol's body for staleness tracking. Pure Go.

### Slot manager

`internal/slot/manager.go` owns `Slot` instances per workspace root. Each `Slot`:

```go
type Slot struct {
    Root       string
    Indexer    *indexer.ProjectIndexer
    QueryFns   *query.Fns
    Cache      *cache.Manager
    IsGit      bool
    StatsFile  string
    CacheGen   atomic.Uint64    // bumped on every index mutation
    Watcher    *watcher.Slot    // optional
    DirMtimes  map[string]int64 // scandir-based incremental update
    mu         sync.RWMutex
}
```

`SwitchProject` is idempotent. Adding/removing roots at runtime is supported via `_register_roots` equivalent.

### Watcher

`internal/watcher/` uses `fsnotify/fsnotify` (pure Go, cross-platform). Replaces Python `watchfiles` (the Rust extension whose SIGSEGV-on-shutdown drove the test-isolation workaround). `TOKEN_SAVIOR_WATCHER=on|auto|off` env var preserved; default `auto` enables when fsnotify can attach, falls back to mtime scan otherwise. The Python `TS_WATCHER_FORCE_POLLING` test escape hatch is no longer needed.

## Query, edit, git

- `internal/query/` — pure functions over `Slot` index state. `find_symbol`, `get_function_source`, `get_class_source`, `get_dependents`, `get_dependencies`, `get_change_impact`, `get_full_context`, `get_call_chain`, `search_codebase`, `get_routes`, `find_hotspots`, `find_dead_code`, `find_semantic_duplicates`, `detect_breaking_changes`, `analyze_config`, `find_impacted_test_files`. Each is a `Handler`-shaped function in `internal/mcp/dispatch.go` calling into `internal/query/*.go`.
- `internal/edit/` — `replace_symbol_source`, `insert_near_symbol`, `add_field_to_model`, `move_symbol`, `edit_lines_in_symbol`. Each bumps `Slot.CacheGen` and triggers re-annotation of the touched file. `add_field_to_model` is restricted to TS files (no Prisma support in v4 — language not in matrix).
- `internal/git/` — `get_git_status`, `get_recent_commits`, `get_blame`, `get_diff_summary`. Uses `os/exec` to shell out to `git` (matches Python).

Session caches: result cache (`_session_result_cache`) and symbol-cache (`_session_symbol_cache`) live on `Slot` as `sync.Map`-backed structures keyed by `(tool, cache_gen, args_hash)`.

## Memory engine — BM25 only

### Storage

`internal/memory/db/` uses **`modernc.org/sqlite`** — pure-Go SQLite with FTS5 support. WAL mode enabled. No `sqlite-vec`, no vector table.

Schema (`internal/memory/db/schema.sql`) mirrors the Python `memory_schema.sql` minus the vec0 virtual table and any embeddings columns. Twelve observation types preserved (`fact`, `note`, `warning`, `convention`, `guardrail`, `research`, `command`, etc.).

### Search

`internal/memory/search/` implements:

- `memory_index` — Layer 1 shortlist (BM25 score, ~15 tok/result).
- `memory_search` — Layer 2 hits (BM25 + decay/access weighting, ~60 tok/result).
- `memory_get` — Layer 3 full record (~200 tok/result).

RRF fusion is unused (no second ranker). Citation URIs (`ts://obs/{id}`) preserved.

### Ranking signals (all kept)

- Bayesian validity prior + update rule (`internal/memory/observations/validity.go`).
- Per-type TTL (command 60d, research 90d, note 60d, …).
- LRU score `0.4·recency + 0.3·access + 0.3·type` (`internal/memory/decay/score.go`).
- Access count × context weight for ROI (`internal/memory/roi/`).
- Auto-promotion (note ×5 → convention, warning ×5 → guardrail).
- MDL distillation grouping (`internal/memory/distill/mdl.go`).
- Contradiction detection at save time, scoped to text similarity over FTS5 (lossier than Python's hybrid version but functional).
- Symbol-staleness: observations linked to symbols are invalidated when the symbol's content-hash changes.

### Auto-extract & viewer

- `TS_AUTO_EXTRACT=1` + `TS_API_KEY` — preserved. PostToolUse hook calls a small-model HTTP endpoint to extract 0-3 observations.
- Web viewer (`TS_VIEWER_PORT`) — keep, but rewrite as Go `net/http` + server-sent events. No htmx. Minimal UI parity with the Python version. **Carve-out:** if the viewer takes more than a week, defer it to v4.1.

### Hooks

Python's 8 Claude Code lifecycle hooks (`hooks/*.sh` + one `.py`) are already mostly bash. The `.sh` hooks ship unchanged. The single `tool_capture_hook.py` is rewritten as a tiny Go binary (`cmd/ts-cli capture-hook`) or as bash that shells out to `ts-cli`. Hook config (`memory-hooks-config.json`, `tool-capture-hooks-config.json`) ships in `hooks/` unchanged.

## Optimization engines (5)

All ported faithfully. Each lives in its own subpackage with the same on-disk format as Python so that v3 → v4 state migration is possible (or noop'd if v3 state is wiped at cutover).

- **`internal/opt/ppm/`** — PPM Markov prefetcher. Trie of counts over tool-call n-grams. Pure numerics; ~300 LoC Go.
- **`internal/opt/tca/`** — Tenseur de Co-Activation. PMI over symbol co-activation events. Pure numerics; ~200 LoC.
- **`internal/opt/leiden/`** — Leiden community detection on the symbol dep graph. Largest engine. Pull `github.com/jbpratt/go-leiden` or hand-port the Python implementation; budget ~600 LoC + tuning.
- **`internal/opt/linucb/`** — LinUCB contextual bandit ranking memory observations for injection. Linear algebra small enough for `gonum/mat` (pure Go, no CGO). ~400 LoC.
- **`internal/opt/warmstart/`** — Session warm-start. Signature-based historical session lookup. ~200 LoC.

All five instantiate at server boot, same as Python's `server_state.py` globals. They write to `~/.local/share/token-savior/` (or `TOKEN_SAVIOR_STATS_DIR` override).

## Test strategy — compatibility harness

### During the port (`cmd/ts-compat`)

The harness:

1. Spins up the Python `token-savior` v3 (kept in-repo during the port) and the Go binary side-by-side.
2. Points both at the same project fixture (`testdata/fixtures/<name>/`).
3. For each tool exercised by the fixture's test script, calls Python and Go with the same arguments.
4. Diffs the structured output. Stateless tools (`find_symbol`, `get_function_source`, `get_dependents`, …) get strict equality. Stateful tools (memory engine writes) get a more permissive shape-compare.
5. Fails on any unjustified diff; expected diffs (e.g., result ordering on equally-ranked items) are listed in `internal/compat/expected_diffs.go` with rationale.

This is the per-milestone fidelity gate. Each milestone's PR must pass the harness on all fixtures.

### CI

The compat harness runs in CI as an additional job (`compat-harness`) until v4.0. It depends on Python being installed in the runner; uses `actions/setup-python@v5` + `pip install -e .[mcp]` and Go side-by-side.

### After cutover (v4.0)

When Python is deleted, the harness mode flips: instead of running Python live, it diffs Go output against `testdata/golden/` (JSON files captured from Python at freeze time). The harness shrinks to a golden-file checker.

`internal/compat/` and `cmd/ts-compat/` get deleted; only the golden files remain. Future regressions show up as golden-file diffs.

### Unit tests in addition

Compat harness covers end-to-end. Inside each Go package, idiomatic `testify` table tests cover edge cases that the harness can't easily exercise (error paths, malformed input, concurrent access). Target ~300-400 hand-written tests across the whole codebase — not a 1:1 port of Python's 1451.

### Test isolation

Equivalent of `tests/conftest.py`: a `TestMain` in each integration test package sets `TOKEN_SAVIOR_STATS_DIR` to a tempdir and `TOKEN_SAVIOR_WATCHER=off`. The watchfiles SIGSEGV that drove the Python workaround doesn't exist for fsnotify, so the watcher can default to `auto` in unit tests.

## Build, CI, distribution

Mirror `bayvets/thepetpanicbutton-triage-server` directly.

### Makefile

Targets ported verbatim where applicable, dropped where they don't apply (no `db-up`, no `db-migrate` — token-savior has no Postgres):

```makefile
build       — go build $(LDFLAGS) -o bin/token-savior ./cmd/token-savior
build-all   — token-savior + ts-cli + ts-compat
build-linux — CGO_ENABLED=0 GOOS=linux GOARCH=amd64/arm64
clean       — rm -rf bin/
test        — go test -race ./... -count=1
test-compat — go run ./cmd/ts-compat (runs compat harness in-process; requires Python venv)
lint        — golangci-lint run ./...
release     — test + docker-build-ci + docker-push
changelog   — git-cliff -o CHANGELOG.md
tag TAG=    — annotated git tag
```

`LDFLAGS` carry `-X main.Version`, `main.Commit`, `main.BuildTime` exactly as in triage-server.

### `.golangci.yml`

Copy from triage-server, drop rules that don't apply (no Postgres-/Stripe-specific exclusions). Keep:

```yaml
linters:
  default: none
  enable:
    - bodyclose
    - errcheck
    - govet
    - ineffassign
    - staticcheck
    - unused
    - gosec
    - prealloc
    - unconvert
    - unparam
```

Run config: `go: '1.26'`, `timeout: 5m`, `tests: true`.

### `.github/workflows/ci.yml`

Five jobs, mirroring triage-server:

- **test** — `go test -race -coverprofile=coverage.out ./...` on Go 1.26.
- **lint** — `golangci-lint-action@v9` with `--timeout=5m`.
- **compat-harness** — sets up Python 3.11 + Go 1.26, installs Python `token-savior` editable, runs `go run ./cmd/ts-compat`. **Deleted at v4.0.**
- **security** — `gosec -no-fail -fmt text ./...`.
- **build** — `GOOS=linux GOARCH=amd64/arm64` static binaries, uploaded as artifacts. Needs [`test`, `lint`, `compat-harness`].

Triage-server's `integration-test` job is dropped (no Postgres dependency).

### `.goreleaser.yaml`

Copy structure from triage-server. One binary (`token-savior`) initially; expand to multi-binary releases when `ts-cli` becomes useful externally. Output: Linux amd64/arm64, Darwin amd64/arm64, Windows amd64. Archives + `goreleaser/dockerfile` for the OCI image.

### Distribution channels

- GitHub Releases via goreleaser.
- `go install token-savior-go/cmd/token-savior@latest` works directly (pure Go, no CGO).
- Homebrew tap (`mibayy/tap/token-savior`) once releases stabilise.
- Existing PyPI `token-savior-recall` package: v3.x stays installable, frozen. No v4 published to PyPI.
- Glama / MCP marketplace listings: update once binaries are public.

## Milestones (Approach A)

Each milestone produces a working `token-savior` binary that passes the compat harness on the milestones-so-far tool set. Time estimates are working-week budgets for a single developer.

| # | Name | Tools delivered | Estimate |
|---|---|---|---|
| **M1** | MCP shell + Go annotator + structural skeleton | `find_symbol`, `get_functions`, `get_classes`, `get_imports`, `search_codebase`, `switch_project`, `list_workspace_roots`, `get_stats`. Go annotator only. Compat harness wired up. | 3–4 wk |
| **M2** | Multi-language annotators + dep graph | Add Java, TS, Rust, shell annotators. Deliver `get_function_source`, `get_class_source`, `get_dependents`, `get_dependencies`, `get_change_impact`, `get_full_context`, `get_call_chain`. | 3–4 wk |
| **M3** | Edit + git + watcher + checkpoints | `replace_symbol_source`, `insert_near_symbol`, `move_symbol`, `edit_lines_in_symbol`, `add_field_to_model` (TS only), `get_git_status`, `get_recent_commits`, `checkpoint_*`, fsnotify watcher. | 3–4 wk |
| **M4** | Memory engine | `memory_save`, `memory_index`, `memory_search`, `memory_get`, `memory_delete`, `memory_admin`, FTS5 storage, Bayesian validity, decay, ROI, MDL distill, contradiction detection at save. | 4–5 wk |
| **M5** | Opt engines + remaining tools | PPM, TCA, Leiden, LinUCB, warm-start. Plus `find_dead_code`, `find_hotspots`, `find_semantic_duplicates`, `detect_breaking_changes`, `analyze_config`, `find_impacted_test_files`, `get_routes`. | 3 wk |
| **M6** | Cutover | Delete Python source + tests. Freeze `testdata/golden/`. Shrink compat harness to golden-file checker. Cut v4.0.0 release via goreleaser. Update README, CHANGELOG, llms-install.md. | 1–2 wk |

**Total budget: 17–22 weeks (~4–5 months).**

The compat harness exists from M1 onward and is the merge gate at every milestone.

**Plan scope:** the first `writing-plans` pass should produce an implementation plan for **M1 only**. M2–M6 each get their own implementation plan, opened at the start of that milestone — this keeps each plan small enough to execute without re-planning mid-stream and lets discoveries in earlier milestones reshape later ones.

## Risks and what kills the schedule

| Risk | Mitigation |
|---|---|
| Hand-rolled tokenizers diverge from tree-sitter behaviour on real-world Java/TS code | Compat harness fixtures include 5+ real OSS projects per language; harness fails on any structural diff. Target: zero unjustified diffs on Spring Boot, Next.js, Tokio. |
| `modernc.org/sqlite` performance is worse than CGO sqlite, hurts memory engine throughput on large stores | Benchmark M4 against Python under a synthetic 50k-observation store. If latency regresses >2x, accept it or switch this single subsystem to `mattn/go-sqlite3` (CGO) before cutover. Documented as a quality gate, not a project blocker. |
| Leiden port turns into a research project | Pull `github.com/jbpratt/go-leiden` or comparable upstream. If no maintained upstream exists, drop Leiden from v4.0 and ship as v4.1. **Schedule slip <2 wk.** |
| Compat harness's Python-side flakes in CI (segfaults from `watchfiles`) | The CI job pins `TOKEN_SAVIOR_WATCHER=off` and `TS_WATCHER_FORCE_POLLING=0` exactly as Python's `conftest.py` already does. |
| `mark3labs/mcp-go` lags MCP spec updates that ship during the port | Pin to a tag; backport schema changes locally if needed. Migrate to upstream once it catches up. |
| Tooling like `ts_search` (semantic tool discovery) feels worse with BM25 | Telemetry: record `ts_search` hit/miss rate post-launch. If miss rate climbs above ~25%, revisit by adding optional embedding sidecar (out of scope for v4.0). |

## What is explicitly out of scope

- 20 non-listed languages: Python, JS (standalone, JSX kept via TS scanner), Ruby, C, C#, JSON, YAML, TOML, INI, Gradle, ENV, XML, HCL, Dockerfile, Prisma, conf, generic, text/Markdown. Their annotators, tests, and any tools that only target them (`add_field_to_model` for `.prisma`, etc.) are removed.
- Vector search, ONNX embeddings, `sqlite-vec`, `fastembed`, `symbol_embeddings`.
- The Python dashboard's htmx UI in its current form (Go viewer is a minimal reimplementation; full UI parity is non-goal for v4.0).
- tsbench parity (192/192) — bench is anchored to Python+Prisma fixtures.
- PyPI v4 publication. Python v3 stays at its current version; the project's distribution channel becomes Go binaries.

## Open questions to resolve in the implementation plan

These are not scope decisions — they're details the `writing-plans` step should pin down:

- Exact MCP SDK version pin and known-gap list.
- Whether `cmd/ts-cli` and `cmd/ts-compat` should be Cobra-based or hand-rolled flag parsing (triage-server uses hand-rolled).
- Whether `Slot` should be a struct or interface; affects mockability.
- Test-fixture project list (which OSS projects in `testdata/fixtures/`).
- Goreleaser config specifics: signing, SBOM, provenance attestation.
- Memory schema version migration plan for users coming from Python v3 SQLite stores (probably: don't migrate; fresh start at v4.0).
- Telemetry channel for `ts_search` miss rate (probably: a counter in `stats.Counters`, dumped to the stats file).

## Appendix — env var compatibility

All Python env vars preserved unless the underlying feature is removed:

| Env var | Status |
|---|---|
| `WORKSPACE_ROOTS` | kept |
| `PROJECT_ROOT` | kept (legacy single-root) |
| `TOKEN_SAVIOR_PROFILE` | kept; profiles `full/core/nav/lean/ultra/tiny/tiny_plus` all kept |
| `TS_MEMORY_DISABLE` | kept |
| `TS_CAPTURE_DISABLED` | kept |
| `TS_HOOK_MINIMAL` | kept |
| `TS_NO_HINTS` | kept |
| `TOKEN_SAVIOR_WATCHER` | kept (`on`/`auto`/`off`); `TS_WATCHER_FORCE_POLLING` retired (fsnotify doesn't need it) |
| `TOKEN_SAVIOR_STATS_DIR` | kept |
| `TOKEN_SAVIOR_MEMORY_AUTO_SAVE` | kept |
| `TS_VIEWER_PORT` | kept; viewer is a Go reimplementation |
| `TS_AUTO_EXTRACT` + `TS_API_KEY` | kept |
| `TELEGRAM_BOT_TOKEN` + `TELEGRAM_CHAT_ID` | kept |

No new env vars introduced at v4.0.
