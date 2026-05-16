#!/usr/bin/env bash
# Captures Python v3 baselines for the M1 exit gate (E3).
#
# Writes testdata/baselines/python-v3-<date>.json with cold-index latency
# and current ts-compat parity status. Re-runs overwrite the same date file.
#
# Override the Python binary via TS_PYTHON_BIN — defaults to "token-savior"
# (PATH lookup). The recommended local setup is a venv:
#   /opt/homebrew/bin/python3 -m venv ~/.venvs/token-savior
#   ~/.venvs/token-savior/bin/pip install -e ".[mcp]"
#   TS_PYTHON_BIN=~/.venvs/token-savior/bin/token-savior ./scripts/capture-baselines.sh

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
FIXTURE="$REPO_ROOT/testdata/fixtures/go-small"
OUT_DIR="$REPO_ROOT/testdata/baselines"
DATE="$(date -u +%Y-%m-%d)"
OUT_FILE="$OUT_DIR/python-v3-$DATE.json"

PYTHON_BIN="${TS_PYTHON_BIN:-token-savior}"

if ! command -v "$PYTHON_BIN" >/dev/null && [ ! -x "$PYTHON_BIN" ]; then
  echo "error: Python token-savior not found at '$PYTHON_BIN'" >&2
  echo "  install via: /opt/homebrew/bin/python3 -m venv ~/.venvs/token-savior" >&2
  echo "               ~/.venvs/token-savior/bin/pip install -e \".[mcp]\"" >&2
  echo "  then re-run: TS_PYTHON_BIN=~/.venvs/token-savior/bin/token-savior $0" >&2
  exit 1
fi

mkdir -p "$OUT_DIR"

echo "Capturing baseline → $OUT_FILE"
echo "  Python: $PYTHON_BIN"
echo "  Fixture: $FIXTURE"

# Cold-boot latency: wall time of one startup until stdin EOF.
# timeout(1) may not exist on macOS without coreutils; fall back gracefully.
_TIMEOUT_CMD=""
if command -v gtimeout >/dev/null 2>&1; then
  _TIMEOUT_CMD="gtimeout 10s"
elif command -v timeout >/dev/null 2>&1; then
  _TIMEOUT_CMD="timeout 10s"
fi
COLD_INDEX_MS=$(
  { /usr/bin/time -p env WORKSPACE_ROOTS="$FIXTURE" $_TIMEOUT_CMD "$PYTHON_BIN" </dev/null; } 2>&1 \
    | awk '/^real / { printf "%.0f", $2 * 1000 }'
) || true

# Run the compat harness once, capture pass/fail. We do NOT abort if it fails
# — the M1 carry-forward (status doc notes #15-16) records why parity is
# expected to be false until T24 resolves the compact-format / field-rename.
TOOLS_PARITY="false"
HARNESS_OUTPUT="$("$REPO_ROOT/bin/ts-compat" -fixture "$FIXTURE" -python "$PYTHON_BIN" 2>&1)" && TOOLS_PARITY="true"

cat > "$OUT_FILE" <<JSON
{
  "captured_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "fixture": "go-small",
  "python_version": "$(python3 --version 2>&1 | awk '{print $2}')",
  "python_bin": "$PYTHON_BIN",
  "cold_index_ms": ${COLD_INDEX_MS:-null},
  "tools_parity": $TOOLS_PARITY
}
JSON

echo "Baseline written to $OUT_FILE"
echo "tools_parity=$TOOLS_PARITY"
