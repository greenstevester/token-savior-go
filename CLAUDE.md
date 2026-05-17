# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Orientation

- The directory is named `token-savior-go` because a Go port (v4) is now in
  progress alongside the shipping Python v3. Python v3 still owns `src/`,
  package `token_savior` (PyPI: `token-savior-recall`); Go v4 lives in
  `cmd/` and `internal/`.
- This is an MCP server. The Python wire entry point is `token-savior`
  (console script in `pyproject.toml` → `token_savior.server:main_sync`).
  The Go binary builds via `make build` to `./bin/token-savior`.
- The optional web viewer entry point is `token-savior-dashboard` (Python).

## Go port (in progress, M1)

A pure-Go rewrite lives alongside the Python source under `cmd/` + `internal/`.
M1 ships 8 tools (`find_symbol`, `get_functions`, `get_classes`, `get_imports`,
`search_codebase`, `switch_project`, `list_workspace_roots`, `get_stats`) with
the Go annotator only. Build with `make build-all`; test with `make test`;
run the compat harness against Python v3 with `make test-compat`. Spec and
plan: `docs/superpowers/specs/` and `docs/superpowers/plans/`.

## Common commands

```bash
# Install (editable, with MCP + dev tools)
pip install -e ".[mcp,dev]"

# Lint (matches CI exactly — see .github/workflows/ci.yml)
ruff check src/ tests/

# Full test suite
pytest tests/ -q

# Single file / single test
pytest tests/test_server.py -q
pytest tests/test_server.py::test_specific_thing -q

# Run the MCP server locally against a project
WORKSPACE_ROOTS=/abs/path/to/project token-savior

# Run benchmarks
pip install -e ".[benchmark]"
token-savior-bench
```

CI runs on Python 3.11 / 3.12 / 3.13. `mypy strict` is configured but not yet wired
into CI — don't be surprised if it flags more than `ruff` does.

## Architecture — the parts you can't infer by browsing

### Handler dispatch — 4 call-shape buckets

`src/token_savior/server_handlers/__init__.py` aggregates per-domain handler dicts
into four buckets, asserted disjoint at import time:

| Bucket            | Signature                                       | Lives in                              |
|-------------------|-------------------------------------------------|---------------------------------------|
| `META_HANDLERS`   | `handler(arguments) -> list[TextContent]`       | stats, project lifecycle, memory admin |
| `MEMORY_HANDLERS` | `handler(arguments) -> str`                     | memory engine (return wrapped by caller) |
| `SLOT_HANDLERS`   | `handler(slot, arguments) -> raw`               | anything needing a project slot       |
| `QFN_HANDLERS`    | `handler(query_fns, arguments) -> raw`          | structural code-navigation queries    |

Adding a tool means: (1) write the handler in the appropriate `server_handlers/*.py`,
(2) add its schema to `tool_schemas.py`, (3) add it to the relevant `HANDLERS`
dict. The disjoint check fails the import if a name appears in two buckets — *don't
silence it*, rename the tool. `tool_schemas.py` is the single source of truth for
the advertised manifest.

### Slot manager — multi-project workspace

`slot_manager.SlotManager` owns one `_ProjectSlot` per workspace root. Each slot
holds its own `ProjectIndexer`, `query_fns`, `CacheManager`, optional `SlotWatcher`,
and a `cache_gen` counter bumped on every index mutation. SLOT_HANDLERS receive the
active slot; META/MEMORY handlers don't. `switch_project` is the idempotent way to
change the active slot. `WORKSPACE_ROOTS` (comma-separated abs paths) drives this;
the legacy `PROJECT_ROOT` single-root mode still works.

### Annotator dispatch — one per language

`annotator.py` maps file extensions to per-language annotators
(`python_annotator.py`, `typescript_annotator.py`, `go_annotator.py`, ~25 total).
Each annotator returns a `StructuralMetadata` and conforms to `AnnotatorProtocol`
(`models.py`). When adding language support: write `<lang>_annotator.py`, register
in `annotator._EXTENSION_MAP`, add `tests/test_markup_<lang>.py` following the
existing pattern. The annotator-protocol contract test
(`tests/test_annotator_protocol.py`) keeps everyone honest.

### Memory engine — its own subpackage

`token_savior/memory/` is a self-contained sub-system: SQLite (WAL + FTS5 +
optional `sqlite-vec`), Bayesian validity, decay/TTL, ROI tracking, MDL
distillation, hybrid BM25+vector search via RRF, optional web viewer. Schema in
`memory_schema.sql`. The progressive-disclosure contract
(`memory_index` → `memory_search` → `memory_get`) is documented in
`docs/progressive-disclosure.md` — keep all three layers consistent if you touch
any of them. Vector search is opt-in via `pip install ".[memory-vector]"`; tests
gracefully skip when `sqlite-vec`/`fastembed` are absent.

### server_state.py — module-level globals

Single source of truth for mutable session state, including the slot manager and
five optimization engines instantiated at import (`PPMPrefetcher`, `TCAEngine`,
`LeidenCommunities`, `LinUCBInjector`, `SessionWarmStart`). Handlers read/write
via `server_state.<name>` so changes propagate across split modules. Don't
re-declare globals in handler modules.

### Profile filtering

`TOKEN_SAVIOR_PROFILE` (env var) trims the *advertised* `tools/list` payload —
handlers stay registered, so a hidden tool still executes if invoked by name.
Valid values: `full` (default), `core`, `nav`, `lean`, `ultra`, `tiny`,
`tiny_plus`. Filter sets live in `server.py` as `_PROFILE_EXCLUDES`. The README
profile table is the source of truth for token-cost math.

> Note: the previous CLAUDE.md used `TS_PROFILE` — that variable name is **not**
> read anywhere in the code. Always use `TOKEN_SAVIOR_PROFILE`.

### Useful env vars (all real, grep `os.environ` in `server.py` / `server_state.py`)

- `WORKSPACE_ROOTS` — comma-separated project roots (canonical)
- `PROJECT_ROOT` — single-root legacy mode
- `TOKEN_SAVIOR_PROFILE` — manifest trimming (see above)
- `TS_MEMORY_DISABLE=1` — hide `memory_*` tools from manifest
- `TS_CAPTURE_DISABLED=1` — hide `capture_*` tools, skip read-side sandbox
- `TS_HOOK_MINIMAL=1` — minimal SessionStart hook
- `TS_NO_HINTS=1` — drop `_hints` / `_suggestion` from tool returns
- `TOKEN_SAVIOR_WATCHER=on|auto|off` — file watcher mode (tests default `off` — see `tests/conftest.py`)
- `TOKEN_SAVIOR_STATS_DIR` — relocate persistent stats (tests isolate to a temp dir)
- `TOKEN_SAVIOR_MEMORY_AUTO_SAVE=1` — enable auto-save tracking
- `TS_VIEWER_PORT` — boot the memory web viewer
- `TS_AUTO_EXTRACT=1` + `TS_API_KEY` — opt-in LLM auto-extraction from tool uses

### Test isolation

`tests/conftest.py` sets `TOKEN_SAVIOR_STATS_DIR` to a fresh tempdir *before*
collection (module-level side effect on import — not a fixture), and pins
`TOKEN_SAVIOR_WATCHER=off`. Reason: the `watchfiles` Rust extension's destructor
has segfaulted at interpreter shutdown on CI runners. Keep the import lazy; tests
that need the watcher set `auto` + `TS_WATCHER_FORCE_POLLING` in their own
autouse fixtures (see `tests/test_watcher.py`).

## Conventions

- New language support: annotator + `_EXTENSION_MAP` entry + `tests/test_markup_*.py`. The protocol contract test catches missing methods automatically.
- New MCP tool: schema in `tool_schemas.py` + handler in correct `server_handlers/` module + entry in the matching `HANDLERS` dict. If you add to the wrong bucket the disjoint assertion fires at import.
- Index-mutating edits: prefer `edit_ops` paths that bump `slot.cache_gen` — bypassing them stales the session result cache (`_session_result_cache`).
- Don't touch the four MCP dispatch tables from outside `server_handlers/`. The shape (`META` / `MEMORY` / `SLOT` / `QFN`) is load-bearing for `call_tool`.

## When Claude Code is working *on this repo*

Token Savior can index its own source. The tool-routing recipe that the prior
CLAUDE.md documented for downstream consumers still applies here:

| Goal | Tool |
|---|---|
| Locate a symbol | `find_symbol(name)` |
| Read a function / class | `get_function_source(name)` / `get_class_source(name)` |
| Orient on a symbol (loc + source + callers + deps) | `get_full_context(name)` |
| Edit Python | `replace_symbol_source` / `insert_near_symbol` (keeps index in sync) |
| Discover other TS tools | `ts_search(query)` |

Native `Edit` / `Write` are fine for `.md`, `.toml`, `.yaml`, `.json`,
`memory_schema.sql`, and CI configs. Prefer the structural editors for `.py`
because they update the index and avoid a full re-annotation.
