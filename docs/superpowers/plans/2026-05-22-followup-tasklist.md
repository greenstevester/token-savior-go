# Token Savior Go Port — Followup Tasklist

> Snapshot of what's outstanding as of session-end 2026-05-22.
> Companion to the M2 kickoff prompt at
> `docs/superpowers/plans/2026-05-17-go-port-m2-kickoff-prompt.md`.

## State at session-end

- **M1 shipped.** PR #1 merged 2026-05-17. 8 MCP tools, Go annotator, compat harness.
- **M1 followup merged.** PR #2 — gitignore + annotator edge-case tests + status sync.
- **M2 plan landed.** PR #3 — `docs/superpowers/plans/2026-05-17-go-port-m2.md` (41 tasks, ~11k lines).
- **M2 kickoff prompt committed** (`b15c376`).
- Main is at `b15c376`. No open PRs. Working tree clean. Local branch only main.

## Immediate — before kicking off M2 execution

- [ ] **Review the M2 plan** at least once end-to-end. Spot-check the trickiest tasks:
  - T6 (TS scanner — brace/template-literal/JSX disambiguation)
  - T10 (JSX/TSX considerations — the classic `<Foo>` generic-vs-tag heuristic)
  - T22 (Java method declarations)
  - T34 (E5 fidelity script)
- [ ] **Decide on the 9 in-plan judgment calls** (full list at the bottom of this file). Either accept-as-is or write a redirect note for the M2 executor before they start.
- [ ] **Confirm M2 sequencing** (shell → TS → Rust → Java → nav tools) still feels right. If something else has come up that should re-order this, now is the time.

## M2 execution (3-4 weeks per spec §467)

- [ ] Open a new Claude Code session. Paste the M2 kickoff prompt verbatim.
- [ ] Branch: `feat/go-port-m2` from main. One PR on completion (matches M1's shape).
- [ ] Stop-checkpoints for personal review: T4 / T12 / T18 / T24 / T33 / T38 / T41.
- [ ] Exit gates to hit: E1 (parity) + E2 (cold-index Go ≤ Python +10%) + E5 (≥95% recall, ≥98% precision per language).

## Post-M2 milestones (each gets its own planning + execution session)

- [ ] **M3 plan** — edit ops (`replace_symbol_source`, `insert_near_symbol`, `move_symbol`, `edit_lines_in_symbol`, `add_field_to_model`) + git ops + fsnotify watcher + checkpoints. 3-4 wk. Exit: E1 + E7 (watcher ≤200ms, zero missed events).
- [ ] **M4 plan** — memory engine (FTS5, Bayesian validity, decay, ROI, MDL distill, contradiction detection at save). 4-5 wk. Exit: E1 + E6 ≥80% recall@10. **Stop-ship if E6 < 60%** (forces reconsidering BM25-only choice).
- [ ] **M5 plan** — optimization engines (PPM, TCA, Leiden, LinUCB, warm-start) + tail tools (`find_dead_code`, `find_hotspots`, `find_semantic_duplicates`, `detect_breaking_changes`, `analyze_config`, `find_impacted_test_files`, `get_routes`). 3 wk. Exit: E1 + E2 + E3 + E4 + E9 all green.
- [ ] **M6 cutover** — delete Python source/tests, freeze `testdata/golden/`, shrink compat harness to golden-file checker, goreleaser v4.0.0, update README/CHANGELOG/llms-install.md. 1-2 wk. Exit: E8 (gobench ≥ Plain Claude) + E10 (2-week dogfood, zero P1s). **Both are stop-ships.**

## Carry-forwards to remember during future milestones

From the M2 plan's 9 in-plan judgment calls — none block M2 acceptance, but they're worth revisiting at M3+ time:

1. **Class{Kind="module"} addition** — currently Rust `mod` is `Kind="alias"`; consider a dedicated `module` kind in M3 model changes.
2. **`Function.Annotations []string`** for Java — annotation names are parsed-but-discarded; M3+ to surface them.
3. **N-ident call-chain parser** — Java `this.owners.findById` collapses to `this.owners`. Extend `parseCalls` or add an `expected_diffs.go` normaliser.
4. **`GetCallChain` `level` parameter** — Python's hop-verbosity knob is M5+ work.
5. **Shell hyphenated alias names** — current POSIX regex; widen if T34 fidelity gate flags any bash-only aliases.
6. **E2 CI hard gate** — T35 writes `gate_status` JSON but T37 doesn't `jq -e` on it. Add an explicit CI assertion if any language exceeds the +10% threshold.

## Optional housekeeping

- [ ] Stale `token-savior` binary at repo root is now gitignored but the file still exists locally. `rm token-savior` if it bothers you.
- [ ] After each future milestone PR merges, consider running `/sg-document-release` (proactive suggestion per the skill's description).
- [ ] Branch protection: PR #3's auto-merge attempt failed with "Auto merge is not allowed for this repository." If you want auto-merge for future milestone PRs, enable it in the repo settings.

## What's deliberately out of scope for v4 (won't ship)

For when someone asks "why doesn't v4 do X":

- Vector search, sqlite-vec, ONNX/fastembed embeddings, semantic memory ranking
- 20+ language annotators from Python v3 (Python, JS, Prisma, Ruby, …)
- htmx dashboard UI
- tsbench 192/192 (the v3 bench includes Python+Prisma which v4 doesn't support)

Documented in spec §16-§31.
