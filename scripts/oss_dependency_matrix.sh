#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SAMPLES_DIR="${1:-/tmp/jkdeps-oss-sources}"
OUT_DIR="${2:-/tmp/jkdeps-oss-deps}"
WORKERS="${WORKERS:-$(getconf _NPROCESSORS_ONLN 2>/dev/null || echo 4)}"
JAVA_GRAMMAR="${JAVA_GRAMMAR:-java20}"
MATRIX_MANIFEST="${OSS_MATRIX_MANIFEST:-$ROOT_DIR/scripts/oss_matrix_targets.txt}"
MAX_ERRORS="${MAX_ERRORS:-20}"
FILE_TIMEOUT="${FILE_TIMEOUT:-0s}"
PROJECT_LIMIT="${PROJECT_LIMIT:-0}"
PROJECT_TIMEOUT="${PROJECT_TIMEOUT:-0}"
FAIL_ON_EXPECTATION="${FAIL_ON_EXPECTATION:-1}"
AUTO_TARGET_ROOT="${OSS_MATRIX_AUTO_TARGET:-0}"
AUTO_TARGET_MAX_FILES="${OSS_MATRIX_AUTO_MAX_FILES:-120}"
MATRIX_COMMAND="${OSS_MATRIX_COMMAND:-deps}"
JKDEPS_BIN="${JKDEPS_BIN:-}"
RESUME_EXISTING="${OSS_MATRIX_RESUME:-1}"
TARGETS_FILE_NAME="selected-targets.csv"

mkdir -p "$OUT_DIR"

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

safe_name() {
  echo "$1" | tr -c 'A-Za-z0-9._-' '_'
}

summary_has_entry() {
  local name="$1"
  [[ -f "$summary" ]] || return 1
  rg -q "^[^,]+,${name//\//\\/}," "$summary"
}

targets_has_entry() {
  local name="$1"
  [[ -f "$targets_file" ]] || return 1
  rg -q "^${name//\//\\/}," "$targets_file"
}

append_target_entry() {
  local name="$1"
  local lang="$2"
  local repo_dir="$3"
  local target_dir="$4"
  local selection_mode="$5"
  local rel_target="$6"
  local include_kts="$7"
  local java_grammar="$8"
  local tmp_file

  tmp_file="$(mktemp)"
  if [[ -f "$targets_file" ]]; then
    rg -v "^${name//\//\\/}," "$targets_file" > "$tmp_file" || true
    mv "$tmp_file" "$targets_file"
  else
    rm -f "$tmp_file"
  fi

  echo "${name},${lang},${repo_dir},${target_dir},${selection_mode},${rel_target},${include_kts},${java_grammar}" >> "$targets_file"
}

discover_source_root() {
  local repo_dir="$1"
  local lang="$2"
  python3 "$ROOT_DIR/scripts/auto_target_selector.py" \
    --repo "$repo_dir" \
    --lang "$lang" \
    --max-files "$AUTO_TARGET_MAX_FILES"
}

extract_int() {
  local file="$1"
  local key="$2"
  rg -o "\"${key}\"[[:space:]]*:[[:space:]]*[0-9]+" "$file" | head -n1 | sed -E "s/.*:[[:space:]]*([0-9]+).*/\\1/" || true
}

extract_float() {
  local file="$1"
  local key="$2"
  rg -o "\"${key}\"[[:space:]]*:[[:space:]]*[0-9]+\\.?[0-9]*" "$file" | head -n1 | sed -E "s/.*:[[:space:]]*([0-9]+\\.?[0-9]*).*/\\1/" || true
}

extract_summary_metric() {
  local file="$1"
  local section="$2"
  local key="$3"
  rg -o "^${section}:[[:space:]]+.*\\b${key}=[0-9]+\\b" "$file" | head -n1 | sed -E "s/.*\\b${key}=([0-9]+)\\b.*/\\1/" || true
}

evaluate_expectations() {
  local name="$1"
  local total_files="$2"
  local parsed_files="$3"
  local failed_files="$4"
  local unresolved_count="$5"
  local min_total_files="$6"
  local min_parsed_files="$7"
  local max_failed_files="$8"
  local max_unresolved="$9"

  local status="pass"
  local reason="ok"

  if [[ -n "$min_total_files" && "$min_total_files" =~ ^[0-9]+$ && "$total_files" -lt "$min_total_files" ]]; then
    status="fail"
    reason="${name}: total_files=${total_files} < min_total_files=${min_total_files}"
  fi

  if [[ "$status" == "pass" && -n "$min_parsed_files" && "$min_parsed_files" =~ ^[0-9]+$ && "$parsed_files" -lt "$min_parsed_files" ]]; then
    status="fail"
    reason="${name}: parsed_files=${parsed_files} < min_parsed_files=${min_parsed_files}"
  fi

  if [[ "$status" == "pass" && -n "$max_failed_files" && "$max_failed_files" =~ ^[0-9]+$ && "$failed_files" -gt "$max_failed_files" ]]; then
    status="fail"
    reason="${name}: failed_files=${failed_files} > max_failed_files=${max_failed_files}"
  fi

  if [[ "$status" == "pass" && -n "$max_unresolved" && "$max_unresolved" =~ ^[0-9]+$ && "$unresolved_count" -gt "$max_unresolved" ]]; then
    status="fail"
    reason="${name}: unresolved=${unresolved_count} > max_unresolved=${max_unresolved}"
  fi

  printf "%s|%s\n" "$status" "$reason"
}

scan_repo() {
  local name="$1"
  local url="$2"
  local ref="$3"
  local target_path="$4"
  local lang="$5"
  local include_kts="$6"
  local project_timeout_override="$7"
  local java_grammar_override="$8"
  local min_total_files="$9"
  local min_parsed_files="${10}"
  local max_failed_files="${11}"
  local max_unresolved="${12}"
  local artifact
  local log
  local time_file
  local result="fail"
  local rc=0
  local include_kts_flag
  local repo_dir
  local target_dir
  local total_files parsed_files failed_files dependency_count unresolved_count duration
  local expectation_status="pass"
  local expectation_detail="not-configured"
  local active_project_timeout="${project_timeout_override:-$PROJECT_TIMEOUT}"
  local active_java_grammar="${java_grammar_override:-$JAVA_GRAMMAR}"
  local -a jkdeps_cmd
  local selection_mode="manifest"
  local relative_target_path="$target_path"

  if [[ -n "$JKDEPS_BIN" ]]; then
    jkdeps_cmd=("$JKDEPS_BIN")
  else
    jkdeps_cmd=(go run ./cmd/jkdeps)
  fi

  repo_dir="$SAMPLES_DIR/$(safe_name "$name")"
  artifact="$OUT_DIR/$(safe_name "$name").deps.json"
  log="$OUT_DIR/$(safe_name "$name").log"
  time_file="$OUT_DIR/$(safe_name "$name").time"

  clone_or_update "$url" "$repo_dir" "$ref"

  if [[ "$AUTO_TARGET_ROOT" == "1" && ( "$target_path" == "." || "$target_path" == "AUTO" ) ]]; then
    target_dir="$(discover_source_root "$repo_dir" "$lang")"
    if [[ -n "$target_dir" ]]; then
      target_path="$(realpath --relative-to="$repo_dir" "$target_dir" 2>/dev/null || echo "$target_dir")"
      relative_target_path="$target_path"
      selection_mode="auto"
      echo "INFO auto target: $name -> $target_path"
    fi
  fi

  target_dir="$repo_dir/$target_path"
  if [[ ! -d "$target_dir" ]]; then
    if [[ "$target_path" != "." ]]; then
      local fallback_target="${target_path#*/}"
      if [[ "$fallback_target" != "$target_path" && -d "$repo_dir/$fallback_target" ]]; then
        echo "WARN fallback target path: $target_path -> $fallback_target (repo=$name)" >&2
        target_dir="$repo_dir/$fallback_target"
        relative_target_path="$fallback_target"
        selection_mode="${selection_mode}+fallback"
      else
        echo "SKIP,$name,$lang,0,0,0,0,0,0,skip,no-such-path:$target_path" | tee -a "$summary"
        echo " - skipped: missing path $target_dir"
        return
      fi
    else
      echo "SKIP,$name,$lang,0,0,0,0,0,0,skip,no-such-path:$target_path" | tee -a "$summary"
      echo " - skipped: missing path $target_dir"
      return
    fi
  fi

  if [[ ! -d "$target_dir" ]]; then
    echo "SKIP,$name,$lang,0,0,0,0,0,0,skip,no-such-path:$target_path" | tee -a "$summary"
    echo " - skipped: missing path $target_dir"
    return
  fi

  append_target_entry "$name" "$lang" "$repo_dir" "$target_dir" "$selection_mode" "$relative_target_path" "$include_kts" "$active_java_grammar"

  if [[ "$RESUME_EXISTING" == "1" ]] && summary_has_entry "$name"; then
    echo "[skip] $name already present in summary"
    return
  fi

  if [[ "$include_kts" == "true" ]]; then
    include_kts_flag="--include-kts=true"
  else
    include_kts_flag="--include-kts=false"
  fi

  if [[ "$MATRIX_COMMAND" == "smoke-parse" ]]; then
    artifact="$OUT_DIR/$(safe_name "$name").parse.txt"
    cmd=(/usr/bin/time -f "%e" -o "$time_file" "${jkdeps_cmd[@]}" smoke-parse \
      --repo "$target_dir" \
      --java-grammar "$active_java_grammar" \
      --workers "$WORKERS" \
      --max-errors "$MAX_ERRORS" \
      --fail-on-error=false \
      "$include_kts_flag")
    if [[ "$FILE_TIMEOUT" != "0" && "$FILE_TIMEOUT" != "0s" ]]; then
      cmd+=(--file-timeout "$FILE_TIMEOUT")
    fi
  else
    cmd=("${jkdeps_cmd[@]}" deps \
      --repo "$target_dir" \
      --java-grammar "$active_java_grammar" \
      --java-parse-mode full \
      --workers "$WORKERS" \
      --max-errors-per-file "$MAX_ERRORS" \
      --lenient \
      --json \
      --out "$artifact" \
      "$include_kts_flag")

    if [[ "$FILE_TIMEOUT" != "0" && "$FILE_TIMEOUT" != "0s" ]]; then
      cmd+=(--file-timeout "$FILE_TIMEOUT")
    fi
  fi

  if [[ "$active_project_timeout" != "0" && "$active_project_timeout" != "" ]]; then
    cmd=(timeout "$active_project_timeout" "${cmd[@]}")
  fi

  if [[ "$MATRIX_COMMAND" == "smoke-parse" ]]; then
    if "${cmd[@]}" >"$artifact" 2>"$log"; then
      result="ok"
    else
      rc=$?
      if [[ "$active_project_timeout" != "0" && "$active_project_timeout" != "" && ( "$rc" -eq 124 || "$rc" -eq 137 ) ]]; then
        result="timeout"
      else
        result="runtime-error"
      fi
    fi
  elif "${cmd[@]}" >/dev/null 2>"$log"; then
    result="ok"
  else
    rc=$?
    if [[ "$active_project_timeout" != "0" && "$active_project_timeout" != "" && ( "$rc" -eq 124 || "$rc" -eq 137 ) ]]; then
      result="timeout"
    else
      result="runtime-error"
    fi
  fi

  if [[ -f "$artifact" && "$MATRIX_COMMAND" == "smoke-parse" ]]; then
    total_files=$(extract_summary_metric "$artifact" Files total)
    parsed_files=$(extract_summary_metric "$artifact" Parse parsed)
    failed_files=$(extract_summary_metric "$artifact" Parse failed)
    dependency_count=0
    unresolved_count=0
    duration="$(tr -d '\r\n' < "$time_file" 2>/dev/null || true)"
  elif [[ -f "$artifact" ]]; then
    total_files=$(extract_int "$artifact" total_files)
    parsed_files=$(extract_int "$artifact" parsed_files)
    failed_files=$(extract_int "$artifact" failed_files)
    dependency_count=$(extract_int "$artifact" dependency_count)
    unresolved_count=$(extract_int "$artifact" unresolved_count)
    duration=$(extract_float "$artifact" duration_seconds)
  else
    total_files=0
    parsed_files=0
    failed_files=0
    dependency_count=0
    unresolved_count=0
    duration=0
  fi

  total_files="${total_files:-0}"
  parsed_files="${parsed_files:-0}"
  failed_files="${failed_files:-0}"
  dependency_count="${dependency_count:-0}"
  unresolved_count="${unresolved_count:-0}"
  duration="${duration:-0}"

  if [[ "$result" == "ok" && -n "$min_total_files$min_parsed_files$max_failed_files$max_unresolved" ]]; then
    IFS='|' read -r expectation_status expectation_detail < <(evaluate_expectations "$name" "$total_files" "$parsed_files" "$failed_files" "$unresolved_count" "$min_total_files" "$min_parsed_files" "$max_failed_files" "$max_unresolved")
  elif [[ "$result" == "ok" ]]; then
    expectation_status="pass"
    expectation_detail="not-configured"
  else
    expectation_status="skip"
    expectation_detail="scan did not complete"
  fi

  expectation_detail="${expectation_detail//,/;}"
  echo "$result,$name,$lang,$total_files,$parsed_files,$failed_files,$dependency_count,$unresolved_count,$duration,$expectation_status,$expectation_detail" >> "$summary"
  echo "[$result] $name ($lang) total=$total_files parsed=$parsed_files failed=$failed_files deps=$dependency_count unresolved=$unresolved_count dur=${duration}s"
}

SUMMARY_FILE="$OUT_DIR/summary.csv"
summary="$SUMMARY_FILE"
targets_file="$OUT_DIR/$TARGETS_FILE_NAME"
if [[ "$RESUME_EXISTING" == "1" && -f "$summary" ]]; then
  :
else
  echo "status,name,language,total_files,parsed_files,failed_files,dependency_count,unresolved_count,duration_seconds,expectation_status,expectation_detail" > "$summary"
fi
if [[ "$RESUME_EXISTING" == "1" && -f "$targets_file" ]]; then
  :
else
  echo "name,language,repo_dir,target_dir,selection_mode,relative_target,include_kts,java_grammar" > "$targets_file"
fi
scanned=0
expectation_failures=0

scan_matrix() {
  local manifest_file="$1"
  while IFS='|' read -r name url ref target_path lang include_kts project_timeout_override java_grammar_override min_total_files min_parsed_files max_failed_files max_unresolved; do
    [[ -z "$name" || "${name:0:1}" == "#" ]] && continue
    scanned=$((scanned + 1))
    if [[ "$PROJECT_LIMIT" != "0" && "$scanned" -gt "$PROJECT_LIMIT" ]]; then
      break
    fi
    scan_repo "$name" "$url" "$ref" "$target_path" "$lang" "$include_kts" "$project_timeout_override" "$java_grammar_override" "$min_total_files" "$min_parsed_files" "$max_failed_files" "$max_unresolved"
  done < "$manifest_file"
}

if [[ -n "${OSS_MATRIX_MANIFEST:-}" && ! -f "$MATRIX_MANIFEST" ]]; then
  echo "manifest missing: $MATRIX_MANIFEST" >&2
  exit 1
fi

if [[ -f "$MATRIX_MANIFEST" ]]; then
  scan_matrix "$MATRIX_MANIFEST"
else
  while IFS='|' read -r name url ref target_path lang include_kts project_timeout_override java_grammar_override min_total_files min_parsed_files max_failed_files max_unresolved; do
    [[ -z "$name" || "${name:0:1}" == "#" ]] && continue
    scanned=$((scanned + 1))
    if [[ "$PROJECT_LIMIT" != "0" && "$scanned" -gt "$PROJECT_LIMIT" ]]; then
      break
    fi
    scan_repo "$name" "$url" "$ref" "$target_path" "$lang" "$include_kts" "$project_timeout_override" "$java_grammar_override" "$min_total_files" "$min_parsed_files" "$max_failed_files" "$max_unresolved"
  done <<'EOF'
# name|url|ref|target_path|lang|include_kts|project_timeout|java_grammar|min_total_files|min_parsed_files|max_failed_files|max_unresolved
# java baseline
google/guava|https://github.com/google/guava.git|9857e70cf51a341ebb41dd2f0b8d3354f6a9d869|guava/src/com/google/common/base|java|false|300|java20|50|50|0|1000
square/okhttp|https://github.com/square/okhttp.git||okhttp/src/jvmMain|java|false|120|java20|1|1|0|200
square/retrofit|https://github.com/square/retrofit.git||retrofit/src/main/java|java|false|300|java20|1|1|0|800
reactivestreams/reactive-streams-jvm|https://github.com/reactive-streams/reactive-streams-jvm.git||.|java|false||java20|1|1|0|800
google/gson|https://github.com/google/gson.git||gson/src/main/java/com/google/gson|java|false|900|java20|1|1|0|1000
apache/commons-io|https://github.com/apache/commons-io.git||src/main/java|java|false|600|java20|1|1|0|2000
apache/commons-codec|https://github.com/apache/commons-codec.git||src/main/java|java|false|600|java20|1|1|0|1500
apache/commons-collections|https://github.com/apache/commons-collections.git||src/main/java/com/google/common/collect|java|false|900|java25|1|1|0|1000
jedis/jedis|https://github.com/redis/jedis.git||src/main/java/redis/clients/jedis|java|false|900|java11|1|1|0|2000
netty/netty|https://github.com/netty/netty.git||transport/src/main/java|java|false|900|java21|1|1|0|4000
# kotlin baseline
Kotlin/kotlinx.coroutines|https://github.com/Kotlin/kotlinx.coroutines.git||kotlinx-coroutines-core/common/src|kotlin|true|||1|1|0|4000
Kotlin/kotlinx.serialization|https://github.com/Kotlin/kotlinx.serialization.git||formats/json/common/src|kotlin|true|||1|1|0|5000
Kotlin/kotlinx-datetime|https://github.com/Kotlin/kotlinx-datetime.git||kotlinx-datetime/common/src|kotlin|true|||1|1|0|5000
Kotlin/kotlinx.atomicfu|https://github.com/Kotlin/kotlinx-atomicfu.git||atomicfu/src/commonMain/kotlin|kotlin|true|||1|1|0|5000
square/okio|https://github.com/square/okio.git||okio/src/commonMain/kotlin|kotlin|true|||1|1|0|5000
arrow-kt/arrow|https://github.com/arrow-kt/arrow.git||arrow-libs/core/arrow-core/src/commonMain/kotlin|kotlin|true|||1|1|0|5000
Koin/koin|https://github.com/InsertKoinIO/koin.git||koin-core/koin-core/src/commonMain/kotlin|kotlin|true|||1|1|0|5000
cashapp/sqldelight|https://github.com/cashapp/sqldelight.git||sqldelight-compiler/src/main/kotlin|kotlin|true|||1|1|0|5000
Kotlin/kotlinx.cli|https://github.com/Kotlin/kotlinx.cli.git||core/src/main/kotlin|kotlin|true|||1|1|0|5000
JetBrains/Exposed|https://github.com/JetBrains/Exposed.git||exposed-core/src/main/kotlin|kotlin|true|||1|1|0|5000
EOF
fi

echo
echo "=== Scan complete ==="
echo "Summary: $SUMMARY_FILE"
cat "$SUMMARY_FILE"
echo "Targets: $targets_file"
if [[ "$FAIL_ON_EXPECTATION" == "1" ]]; then
  scan_failures="$(rg -c '^(runtime-error|timeout|skip|fail),' "$summary" || true)"
  if [[ "$scan_failures" -gt 0 ]]; then
    echo "FAIL: scan failed for $scan_failures row(s). Set FAIL_ON_EXPECTATION=0 to skip this check." >&2
    exit 2
  fi
  expectation_failures="$(rg -c ',fail,' "$summary" || true)"
  if [[ "$expectation_failures" -gt 0 ]]; then
    echo "FAIL: expectation check failed for $expectation_failures row(s). Set FAIL_ON_EXPECTATION=0 to skip this check." >&2
    exit 2
  fi
fi
