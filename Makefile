.PHONY: generate smoke smoke-stress smoke-stress-strict porting-acceptance porting-acceptance-stress porting-acceptance-strict porting-acceptance-core porting-acceptance-full verify-porting-completion verify-porting-completion-json compare-porting-baseline update-porting-baseline check-porting-baseline-sync porting-audit check-baseline-label-gate test-porting-gates porting-complete jkdeps-graph kcg-parse kcg-symbols kcg-deps kcg-resolve kcg-graph kcg-acceptance kcg-official-parity kcg-resolve-runtime kcg-inventory kcg-inventory-js kcg-inventory-browser kcg-inventory-atomicfu kcg-inventory-runtime oss-matrix

oss-matrix:
	./scripts/oss_dependency_matrix.sh

PORTING_SAMPLES_DIR ?= /tmp/jkdeps-samples
PORTING_OUT_DIR ?= /tmp/jkdeps-porting
BASELINE_FILE_PATH ?= docs/porting-baseline.json
KOTLIN_OFFICIAL_PARITY_MAX_MISSING_IN_GO ?= 0
KOTLIN_OFFICIAL_PARITY_MAX_MISSING_IN_OFFICIAL ?= 0
KOTLIN_OFFICIAL_PARITY_MAX_PARSE_STATUS_MISMATCH ?= 0
KOTLIN_OFFICIAL_PARITY_MAX_PACKAGE_MISMATCH ?= 0
KOTLIN_OFFICIAL_PARITY_MAX_IMPORT_MISMATCH ?= 0
KOTLIN_OFFICIAL_PARITY_MAX_DECLARATION_MISMATCH ?= 0

generate:
	./scripts/generate_parsers.sh

smoke:
	./scripts/smoke_parse_samples.sh

smoke-stress:
	RUN_GUAVA_STRESS=1 ./scripts/smoke_parse_samples.sh

smoke-stress-strict:
	RUN_GUAVA_STRESS_STRICT=1 ./scripts/smoke_parse_samples.sh

porting-acceptance:
	./scripts/porting_acceptance.sh "$(PORTING_SAMPLES_DIR)" "$(PORTING_OUT_DIR)"

porting-acceptance-stress:
	RUN_GUAVA_STRESS=1 ./scripts/porting_acceptance.sh "$(PORTING_SAMPLES_DIR)" "$(PORTING_OUT_DIR)"

porting-acceptance-strict:
	RUN_GUAVA_STRESS_STRICT=1 ./scripts/porting_acceptance.sh "$(PORTING_SAMPLES_DIR)" "$(PORTING_OUT_DIR)"

porting-acceptance-core:
	RUN_KOTLIN_CORE=1 ./scripts/porting_acceptance.sh "$(PORTING_SAMPLES_DIR)" "$(PORTING_OUT_DIR)"

porting-acceptance-full:
	RUN_GUAVA_STRESS_STRICT=1 RUN_KOTLIN_JVM=1 RUN_KOTLIN_CORE=1 RUN_KOTLIN_OFFICIAL_PARITY=1 KOTLIN_OFFICIAL_PARITY_MAX_MISSING_IN_GO="$(KOTLIN_OFFICIAL_PARITY_MAX_MISSING_IN_GO)" KOTLIN_OFFICIAL_PARITY_MAX_MISSING_IN_OFFICIAL="$(KOTLIN_OFFICIAL_PARITY_MAX_MISSING_IN_OFFICIAL)" KOTLIN_OFFICIAL_PARITY_MAX_PARSE_STATUS_MISMATCH="$(KOTLIN_OFFICIAL_PARITY_MAX_PARSE_STATUS_MISMATCH)" KOTLIN_OFFICIAL_PARITY_MAX_PACKAGE_MISMATCH="$(KOTLIN_OFFICIAL_PARITY_MAX_PACKAGE_MISMATCH)" KOTLIN_OFFICIAL_PARITY_MAX_IMPORT_MISMATCH="$(KOTLIN_OFFICIAL_PARITY_MAX_IMPORT_MISMATCH)" KOTLIN_OFFICIAL_PARITY_MAX_DECLARATION_MISMATCH="$(KOTLIN_OFFICIAL_PARITY_MAX_DECLARATION_MISMATCH)" ./scripts/porting_acceptance.sh "$(PORTING_SAMPLES_DIR)" "$(PORTING_OUT_DIR)"

verify-porting-completion:
	./scripts/verify_porting_completion.sh "$(PORTING_OUT_DIR)"

verify-porting-completion-json:
	./scripts/verify_porting_completion.sh "$(PORTING_OUT_DIR)" --json-out "$(PORTING_OUT_DIR)/porting-completion-verify.json"

compare-porting-baseline:
	./scripts/compare_porting_baseline.sh "$(PORTING_OUT_DIR)" "$(BASELINE_FILE_PATH)"

update-porting-baseline:
	./scripts/update_porting_baseline.sh "$(PORTING_OUT_DIR)" "$(BASELINE_FILE_PATH)"

check-porting-baseline-sync:
	./scripts/update_porting_baseline.sh "$(PORTING_OUT_DIR)" "$(BASELINE_FILE_PATH)" --check

porting-audit:
	./scripts/porting_audit.sh "$(PORTING_OUT_DIR)" "$(BASELINE_FILE_PATH)" "$(PORTING_OUT_DIR)/porting-audit.json"

check-baseline-label-gate:
	./scripts/check_baseline_label_gate.sh --event-name pull_request --base-sha "$${BASE_SHA:?set BASE_SHA}" --labels "$${PR_LABELS:-}" --required-label "$${BASELINE_APPROVAL_LABEL:-baseline-approved}" --baseline-file "$(BASELINE_FILE_PATH)" --json-out /tmp/jkdeps-baseline-label-gate.json

test-porting-gates:
	go test ./internal/porting

porting-complete: porting-acceptance-full porting-audit

jkdeps-graph:
	go run ./cmd/jkdeps graph --repo . --java-grammar java20 --group-by package --inventory ./kotlin-compiler-golang/inventory/runtime-index.json --lenient --out ./jkdeps-mixed-graph

kcg-parse:
	go run ./cmd/kotlin-compiler-golang parse --repo .

kcg-symbols:
	go run ./cmd/kotlin-compiler-golang symbols --repo .

kcg-deps:
	go run ./cmd/kotlin-compiler-golang deps --repo .

kcg-resolve:
	go run ./cmd/kotlin-compiler-golang resolve --repo .

kcg-graph:
	go run ./cmd/kotlin-compiler-golang graph --repo . --inventory ./kotlin-compiler-golang/inventory/runtime-index.json --lenient --out ./jkdeps-graph

kcg-acceptance:
	go run ./cmd/kotlin-compiler-golang acceptance --repo . --inventory ./kotlin-compiler-golang/inventory/runtime-index.json --lenient --max-failed-files 0 --max-unresolved-imports 0 --out ./kcg-acceptance.json

kcg-official-parity:
	MAX_MISSING_IN_GO="$(KOTLIN_OFFICIAL_PARITY_MAX_MISSING_IN_GO)" MAX_MISSING_IN_OFFICIAL="$(KOTLIN_OFFICIAL_PARITY_MAX_MISSING_IN_OFFICIAL)" MAX_PARSE_STATUS_MISMATCH="$(KOTLIN_OFFICIAL_PARITY_MAX_PARSE_STATUS_MISMATCH)" MAX_PACKAGE_MISMATCH="$(KOTLIN_OFFICIAL_PARITY_MAX_PACKAGE_MISMATCH)" MAX_IMPORT_MISMATCH="$(KOTLIN_OFFICIAL_PARITY_MAX_IMPORT_MISMATCH)" MAX_DECLARATION_MISMATCH="$(KOTLIN_OFFICIAL_PARITY_MAX_DECLARATION_MISMATCH)" ./scripts/kcg_official_parity.sh "$(PORTING_SAMPLES_DIR)/kotlinx.coroutines/kotlinx-coroutines-core/js/src" "$(PORTING_OUT_DIR)" "$(PORTING_OUT_DIR)/kotlin-official-parity.json"

kcg-resolve-runtime:
	go run ./cmd/kotlin-compiler-golang resolve --repo . --inventory ./kotlin-compiler-golang/inventory/runtime-index.json --lenient

kcg-inventory:
	./scripts/build_kcg_inventory.sh

kcg-inventory-js:
	./scripts/build_kcg_stdlib_js_inventory.sh

kcg-inventory-browser:
	./scripts/build_kcg_browser_inventory.sh

kcg-inventory-atomicfu:
	./scripts/build_kcg_atomicfu_inventory.sh

kcg-inventory-runtime:
	./scripts/build_kcg_runtime_inventory.sh
