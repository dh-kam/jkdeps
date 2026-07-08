package kotlincompilergolang

import "testing"

func TestDebugCandidateMatrix(t *testing.T) {
	files := []string{
		debugSamplePath("kotlinx.coroutines", "kotlinx-coroutines-core", "jvm", "test", "EventLoopsTest.kt"),
		debugSamplePath("kotlinx.coroutines", "kotlinx-coroutines-core", "jvm", "test", "exceptions", "StackTraceRecoveryTest.kt"),
	}
	c := New(Config{Workers: 1, MaxErrorsPerFile: 50, IncludeKTS: true})
	for _, path := range files {
		src := readSampleSourceOrSkip(t, path)
		n := normalizeKotlinSourceForANTLR(src)
		cand := []struct {
			name string
			src  []byte
		}{
			{"norm", n},
			{"paren", normalizeParenthesizedLambdaBodies(n)},
			{"known", normalizeKnownTrailingLambdaCalls(n)},
			{"known+paren", normalizeParenthesizedLambdaBodies(normalizeKnownTrailingLambdaCalls(n))},
			{"generic", normalizeGenericTrailingLambdaCalls(n)},
			{"generic+paren", normalizeParenthesizedLambdaBodies(normalizeGenericTrailingLambdaCalls(n))},
			{"generic-simple-runTest", normalizeGenericTrailingLambdaCallsSimpleRunTest(n)},
			{"generic-simple-runTest+paren", normalizeParenthesizedLambdaBodies(normalizeGenericTrailingLambdaCallsSimpleRunTest(n))},
			{"generic-no-runTest", normalizeGenericTrailingLambdaCallsWithoutRunTest(n)},
			{"generic-no-runTest+paren", normalizeParenthesizedLambdaBodies(normalizeGenericTrailingLambdaCallsWithoutRunTest(n))},
			{"deleg", normalizeDelegationConstructorsAndUnsignedLiterals(n)},
			{"deleg+generic", normalizeGenericTrailingLambdaCalls(normalizeDelegationConstructorsAndUnsignedLiterals(n))},
			{"deleg+generic+paren", normalizeParenthesizedLambdaBodies(normalizeGenericTrailingLambdaCalls(normalizeDelegationConstructorsAndUnsignedLiterals(n)))},
			{"deleg+generic-simple-runTest", normalizeGenericTrailingLambdaCallsSimpleRunTest(normalizeDelegationConstructorsAndUnsignedLiterals(n))},
			{"deleg+generic-simple-runTest+paren", normalizeParenthesizedLambdaBodies(normalizeGenericTrailingLambdaCallsSimpleRunTest(normalizeDelegationConstructorsAndUnsignedLiterals(n)))},
			{"deleg+generic-no-runTest", normalizeGenericTrailingLambdaCallsWithoutRunTest(normalizeDelegationConstructorsAndUnsignedLiterals(n))},
			{"deleg+generic-no-runTest+paren", normalizeParenthesizedLambdaBodies(normalizeGenericTrailingLambdaCallsWithoutRunTest(normalizeDelegationConstructorsAndUnsignedLiterals(n)))},
		}
		t.Logf("=== %s ===", path)
		for _, cc := range cand {
			out := c.parseWithRule(path, cc.src, false)
			t.Logf("%s: diag=%d", cc.name, len(out.diagnostics))
			for i, d := range out.diagnostics {
				if i >= 2 {
					break
				}
				t.Logf("  %d) l=%d c=%d %s", i+1, d.Line, d.Column, d.Message)
			}
		}
	}
}
