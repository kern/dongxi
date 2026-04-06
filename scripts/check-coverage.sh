#!/usr/bin/env bash
set -euo pipefail

# check-coverage.sh — Enforce 100% test coverage with an exclusion list.
#
# Every function must be at 100% coverage unless it appears in the exclusion
# list below. Each exclusion must have a reason comment. If you add a new
# exclusion, explain why it cannot be tested.
#
# Usage:
#   ./scripts/check-coverage.sh [coverage.out]
#
# If no coverage file is provided, one is generated via `go test`.

# ── Exclusion list ────────────────────────────────────────────────────────────
# Format: "file:function min_pct"
# Keep sorted by package then function.

EXCLUSIONS=(
  # Production wiring: calls os.Exit, cannot be captured in-process.
  "github.com/kern/dongxi/cmd/root.go:Execute 0.0"

  # Thin wrapper that calls cmd.Execute(); same os.Exit issue.
  "github.com/kern/dongxi/main.go:main 0.0"

  # Production wiring: reads real config file and calls real API client.
  "github.com/kern/dongxi/cmd/state.go:LoadState 0.0"

  # Dead-code defensive branches inside csv.Writer (buffered; Write never
  # errors) and unreachable return after validated format switch.
  "github.com/kern/dongxi/cmd/export.go:runExport 96.8"
  "github.com/kern/dongxi/cmd/export.go:writeCSV 90.0"

  # Dead-code default case after validOps pre-check, and nil-guard on
  # resolved UUID that can never be nil.
  "github.com/kern/dongxi/cmd/batch.go:runBatch 98.8"

  # Panic branch on rand.Read failure — intentionally untestable.
  "github.com/kern/dongxi/cmd/create.go:newUUID 94.4"

  # Config save after successful server reset — requires real filesystem
  # config created by `dongxi login`.
  "github.com/kern/dongxi/cmd/reset.go:runReset 89.0"

  # SaveConfig: error path on os.MkdirAll / os.WriteFile with real filesystem.
  "github.com/kern/dongxi/dongxi/config.go:SaveConfig 91.7"
)

# ── Helpers ───────────────────────────────────────────────────────────────────

# Look up a key in EXCLUSIONS. Prints the min_pct if found, empty string if not.
lookup_exclusion() {
  local needle="$1"
  for entry in "${EXCLUSIONS[@]}"; do
    local key="${entry% *}"
    local min="${entry##* }"
    if [[ "$key" == "$needle" ]]; then
      echo "$min"
      return
    fi
  done
  echo ""
}

# ── Main ──────────────────────────────────────────────────────────────────────

COVERFILE="${1:-}"
GENERATED=0

if [[ -z "$COVERFILE" ]]; then
  COVERFILE="$(mktemp)"
  GENERATED=1
  echo "Generating coverage profile..."
  go test ./... -coverprofile="$COVERFILE" -covermode=atomic -count=1
fi

if [[ ! -f "$COVERFILE" ]]; then
  echo "ERROR: coverage file not found: $COVERFILE"
  exit 1
fi

# Parse `go tool cover -func` output and check each function.
# Output format per line:
#   github.com/kern/dongxi/cmd/root.go:23:	Execute		0.0%
# The location is "pkg/file.go:linenum:" — we need "pkg/file.go:funcName" as key.

TMPFAILS="$(mktemp)"
cleanup() { rm -f "$TMPFAILS"; [[ "$GENERATED" -eq 1 ]] && rm -f "$COVERFILE"; true; }
trap cleanup EXIT

go tool cover -func="$COVERFILE" | while IFS= read -r line; do
  # Skip the total line
  case "$line" in
    total:*) continue ;;
  esac

  # Extract fields
  location=$(echo "$line" | awk '{print $1}')
  func_name=$(echo "$line" | awk '{print $2}')
  pct=$(echo "$line" | awk '{print $NF}' | tr -d '%')

  if [[ "$pct" == "100.0" ]]; then
    continue
  fi

  # location = "pkg/file.go:linenum:" — strip trailing colon and line number
  location="${location%:}"       # remove trailing colon → pkg/file.go:linenum
  file="${location%:*}"          # remove :linenum → pkg/file.go
  key="${file}:${func_name}"

  min=$(lookup_exclusion "$key")

  if [[ -n "$min" ]]; then
    if awk "BEGIN{exit (!($pct >= $min))}"; then
      continue
    fi
    echo "FAIL: $key coverage ${pct}% dropped below excluded minimum ${min}%"
    echo "1" >> "$TMPFAILS"
  else
    echo "FAIL: $key coverage ${pct}% (not 100% and not in exclusion list)"
    echo "1" >> "$TMPFAILS"
  fi
done

if [[ -s "$TMPFAILS" ]]; then
  echo ""
  echo "Coverage check failed."
  echo "If a function genuinely cannot reach 100%, add it to the EXCLUSIONS"
  echo "list in scripts/check-coverage.sh with a reason comment."
  exit 1
fi

echo "Coverage check passed — all functions at 100% or excluded with justification."
