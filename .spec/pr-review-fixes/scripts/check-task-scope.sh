#!/usr/bin/env bash
set -euo pipefail

TASK_ID="${1:-}"
FEATURE_DIR=".spec/pr-review-fixes"
SCOPE_FILE="$FEATURE_DIR/task-scope.json"

if [[ -z "$TASK_ID" ]]; then
  echo "usage: $0 <TASK-ID>" >&2
  exit 2
fi

if [[ ! -f "$SCOPE_FILE" ]]; then
  echo "scope file not found: $SCOPE_FILE" >&2
  exit 2
fi

if ! command -v jq >/dev/null 2>&1; then
  echo "jq is required for task scope checks" >&2
  exit 2
fi

if ! jq -e --arg task "$TASK_ID" '.tasks[$task]' "$SCOPE_FILE" >/dev/null; then
  echo "unknown TASK: $TASK_ID" >&2
  exit 2
fi

changed=()
while IFS= read -r file; do
  [[ -n "$file" ]] && changed+=("$file")
done < <(
  {
    git diff --name-only
    git ls-files --others --exclude-standard
  } | sort -u
)

if [[ ${#changed[@]} -eq 0 ]]; then
  echo "No changed files."
  exit 0
fi

allowed=()
while IFS= read -r pattern; do
  [[ -n "$pattern" ]] && allowed+=("$pattern")
done < <(jq -r --arg task "$TASK_ID" '.tasks[$task].allowedFiles[]' "$SCOPE_FILE")

forbidden=()
while IFS= read -r pattern; do
  [[ -n "$pattern" ]] && forbidden+=("$pattern")
done < <(jq -r --arg task "$TASK_ID" '.tasks[$task].forbiddenFiles[]?' "$SCOPE_FILE")

matches_pattern() {
  local file="$1"
  local pattern="$2"
  case "$file" in
    $pattern) return 0 ;;
    *) return 1 ;;
  esac
}

out_of_scope=()
for file in "${changed[@]}"; do
  [[ "$file" == "$FEATURE_DIR/"* ]] && continue

  for pattern in "${forbidden[@]}"; do
    if matches_pattern "$file" "$pattern"; then
      out_of_scope+=("$file (forbidden by $pattern)")
      continue 2
    fi
  done

  matched=false
  for pattern in "${allowed[@]}"; do
    if matches_pattern "$file" "$pattern"; then
      matched=true
      break
    fi
  done

  if [[ "$matched" != true ]]; then
    out_of_scope+=("$file")
  fi
done

if [[ ${#out_of_scope[@]} -gt 0 ]]; then
  echo "Out-of-scope files for $TASK_ID:" >&2
  printf '  %s\n' "${out_of_scope[@]}" >&2
  exit 1
fi

echo "Scope check passed for $TASK_ID."
