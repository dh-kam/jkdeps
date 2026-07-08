package kotlincompilergolang

import (
	"bufio"
	"fmt"
	"strings"
	"testing"
)

func dumpLineRange(t *testing.T, label string, src []byte, from, to int) {
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

func TestDebugCoreHotspots(t *testing.T) {
	targets := []struct {
		path  string
		lines []int
	}{
		{debugSamplePath("kotlinx.coroutines", "kotlinx-coroutines-core", "nativeDarwin", "src", "Dispatchers.kt"), []int{20, 79}},
		{debugSamplePath("kotlinx.coroutines", "kotlinx-coroutines-core", "jvm", "test", "EventLoopsTest.kt"), []int{72}},
		{debugSamplePath("kotlinx.coroutines", "kotlinx-coroutines-core", "common", "test", "channels", "ChannelReceiveCatchingTest.kt"), []int{90}},
		{debugSamplePath("kotlinx.coroutines", "kotlinx-coroutines-core", "jvm", "test", "exceptions", "StackTraceRecoveryTest.kt"), []int{214}},
	}
	c := New(Config{Workers: 1, MaxErrorsPerFile: 50, IncludeKTS: true})
	for _, tg := range targets {
		src := readSampleSourceOrSkip(t, tg.path)
		norm := normalizeKotlinSourceForANTLR(src)
		paren := normalizeParenthesizedLambdaBodies(norm)
		trail := normalizeKnownTrailingLambdaCalls(norm)
		both := normalizeParenthesizedLambdaBodies(trail)
		t.Logf("=== %s ===", tg.path)
		outNorm := c.parseWithRule(tg.path, norm, false)
		outParen := c.parseWithRule(tg.path, paren, false)
		outTrail := c.parseWithRule(tg.path, trail, false)
		outBoth := c.parseWithRule(tg.path, both, false)
		t.Logf("diag counts norm=%d paren=%d trail=%d both=%d", len(outNorm.diagnostics), len(outParen.diagnostics), len(outTrail.diagnostics), len(outBoth.diagnostics))
		for i, d := range outNorm.diagnostics {
			if i >= 6 {
				break
			}
			t.Logf(" norm %d) l=%d c=%d %s", i+1, d.Line, d.Column, d.Message)
		}
		for _, ln := range tg.lines {
			from := ln - 4
			if from < 1 {
				from = 1
			}
			to := ln + 6
			dumpLineRange(t, fmt.Sprintf("orig around %d", ln), src, from, to)
			dumpLineRange(t, fmt.Sprintf("norm around %d", ln), norm, from, to)
			dumpLineRange(t, fmt.Sprintf("paren around %d", ln), paren, from, to)
			dumpLineRange(t, fmt.Sprintf("trail around %d", ln), trail, from, to)
			dumpLineRange(t, fmt.Sprintf("both around %d", ln), both, from, to)
		}
	}
}
