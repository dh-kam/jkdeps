#!/usr/bin/env bash
set -euo pipefail

DEST_ROOT="${1:-/tmp/jkdeps-samples}"
mkdir -p "$DEST_ROOT"

DEFAULT_GUAVA_REF="9857e70cf51a341ebb41dd2f0b8d3354f6a9d869"
DEFAULT_KOTLINX_COROUTINES_REF="b11abdf01d4d5db85247ab365abc72efc7b95062"
GUAVA_REF="${GUAVA_REF:-$DEFAULT_GUAVA_REF}"
KOTLINX_COROUTINES_REF="${KOTLINX_COROUTINES_REF:-$DEFAULT_KOTLINX_COROUTINES_REF}"

clone_or_update() {
  local url="$1"
  local dir="$2"
  local ref="${3:-}"

  if [[ -d "$dir/.git" ]]; then
    if [[ -n "$ref" ]]; then
      git -C "$dir" fetch --depth 1 origin "$ref"
      git -C "$dir" checkout -q FETCH_HEAD
    else
      git -C "$dir" fetch --depth 1 origin
      git -C "$dir" checkout -q FETCH_HEAD
    fi
  else
    git clone --depth 1 "$url" "$dir"
    if [[ -n "$ref" ]]; then
      git -C "$dir" fetch --depth 1 origin "$ref"
      git -C "$dir" checkout -q FETCH_HEAD
    fi
  fi
}

clone_or_update "https://github.com/google/guava.git" "$DEST_ROOT/guava" "$GUAVA_REF"
clone_or_update "https://github.com/Kotlin/kotlinx.coroutines.git" "$DEST_ROOT/kotlinx.coroutines" "$KOTLINX_COROUTINES_REF"

echo "Sample repositories are ready in $DEST_ROOT"
echo "guava: $(git -C "$DEST_ROOT/guava" rev-parse HEAD) (ref=$GUAVA_REF)"
echo "kotlinx.coroutines: $(git -C "$DEST_ROOT/kotlinx.coroutines" rev-parse HEAD) (ref=$KOTLINX_COROUTINES_REF)"
