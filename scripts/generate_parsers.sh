#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
JAR_PATH="$($ROOT_DIR/scripts/download_antlr.sh "$ROOT_DIR/tools")"

generate() {
  local output_dir="$1"
  local package_name="$2"
  shift 2

  mkdir -p "$output_dir"
  java -jar "$JAR_PATH" \
    -Dlanguage=Go \
    -encoding UTF-8 \
    -listener \
    -no-visitor \
    -Xexact-output-dir \
    -package "$package_name" \
    -o "$output_dir" \
    "$@"
}

generate "$ROOT_DIR/internal/parsers/javaorig" "javaorig" \
  "$ROOT_DIR/grammars/java/java/JavaLexer.g4" \
  "$ROOT_DIR/grammars/java/java/JavaParser.g4"

# Work around Go target emission in this grammar where semantic predicates call `this.*`.
sed -i \
  -e 's/this.IsNotIdentifierAssign()/p.IsNotIdentifierAssign()/g' \
  -e 's/this.DoLastRecordComponent()/p.DoLastRecordComponent()/g' \
  "$ROOT_DIR/internal/parsers/javaorig/java_parser.go"

generate "$ROOT_DIR/internal/parsers/java8" "java8" \
  "$ROOT_DIR/grammars/java/java8/Java8Lexer.g4" \
  "$ROOT_DIR/grammars/java/java8/Java8Parser.g4"

generate "$ROOT_DIR/internal/parsers/java9" "java9" \
  "$ROOT_DIR/grammars/java/java9/Java9Lexer.g4" \
  "$ROOT_DIR/grammars/java/java9/Java9Parser.g4"

generate "$ROOT_DIR/internal/parsers/java20" "java20" \
  "$ROOT_DIR/grammars/java/java20/Java20Lexer.g4" \
  "$ROOT_DIR/grammars/java/java20/Java20Parser.g4"

generate "$ROOT_DIR/internal/parsers/kotlin" "kotlin" \
  "$ROOT_DIR/grammars/kotlin/UnicodeClasses.g4" \
  "$ROOT_DIR/grammars/kotlin/KotlinLexer.g4" \
  "$ROOT_DIR/grammars/kotlin/KotlinParser.g4"

echo "Generated parsers under internal/parsers"
