# Parser Validation Notes

## Java grammar version coverage

Source set: `/tmp/jkdeps-version-sample` (5 Java files from Guava)

- `--java-grammar java`: parsed 5/5
- `--java-grammar java8`: parsed 5/5
- `--java-grammar java9`: parsed 5/5
- `--java-grammar java20`: parsed 5/5

## OSS smoke parsing

### Java-heavy module

Command:

```bash
go run ./cmd/jkdeps smoke-parse \
  --repo /tmp/jkdeps-samples/guava/guava/src/com/google/common/base \
  --java-grammar java20 \
  --file-timeout 2s \
  --workers 4
```

Result:

- total: 53
- parsed: 53
- failed: 0
- success: 100%

Optional stress command:

```bash
go run ./cmd/jkdeps smoke-parse \
  --repo /tmp/jkdeps-samples/guava/guava/src/com/google/common/util/concurrent \
  --java-grammar java20 \
  --workers 4 \
  --file-timeout 3s \
  --fail-on-error=false
```

### Kotlin-heavy module

Command:

```bash
go run ./cmd/jkdeps smoke-parse \
  --repo /tmp/jkdeps-samples/kotlinx.coroutines/kotlinx-coroutines-core/common/src \
  --java-grammar java20 \
  --workers 4
```

Result:

- total: 111
- parsed: 78
- failed: 33
- success: 70.27%

Representative failures are driven by modern Kotlin constructs that the current `grammars-v4` Kotlin grammar does not fully support (for example `expect` declarations and some inline/value patterns).

## Header round-trip harness

Minimal validation command:

```bash
go run ./cmd/jkdeps roundtrip-check \
  --repo ./testdata/samples \
  --workers 1 \
  --include-kts=true \
  --limit 8
```

Current behavior:

- The harness rebuilds only the parsed `package`/`import` header.
- If no formatter command is configured, files that require rewritten-source normalization are reported as `unsupported` instead of noisy `diff`.
- Formatter hooks are available through `--java-format-cmd` and `--kotlin-format-cmd` using a `{file}` placeholder for the temporary rewritten source path.
