package kotlincompilergolang

import (
	"bufio"
	"strings"
	"testing"
)

func dumpRange2(t *testing.T, label string, src []byte, from, to int) {
	t.Logf("--- %s %d..%d ---", label, from, to)
	s := bufio.NewScanner(strings.NewReader(string(src)))
	ln := 1
	for s.Scan() {
		if ln >= from && ln <= to {
			t.Logf("%4d: %s", ln, s.Text())
		}
		ln++
	}
}

func TestDebugGenericDump(t *testing.T) {
	files := []string{
		debugSamplePath("kotlinx.coroutines", "kotlinx-coroutines-core", "jvm", "test", "EventLoopsTest.kt"),
		debugSamplePath("kotlinx.coroutines", "kotlinx-coroutines-core", "jvm", "test", "exceptions", "StackTraceRecoveryTest.kt"),
	}
	for _, path := range files {
		src := readSampleSourceOrSkip(t, path)
		n := normalizeKotlinSourceForANTLR(src)
		g := normalizeGenericTrailingLambdaCalls(n)
		t.Logf("=== %s ===", path)
		dumpRange2(t, "orig", src, 12, 35)
		dumpRange2(t, "generic", g, 12, 35)
		dumpRange2(t, "orig2", src, 70, 110)
		dumpRange2(t, "generic2", g, 70, 110)
	}
}
