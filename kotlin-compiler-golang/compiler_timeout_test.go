package kotlincompilergolang

import (
	"strings"
	"testing"
	"time"
)

func TestParseOutcomeWithTimeout_NoTimeout(t *testing.T) {
	expected := parseSourceOutcome{
		diagnostics: []Diagnostic{{
			Path:     "/tmp/A.kt",
			Message:  "ok",
			Severity: SeverityError,
		}},
	}

	outcome := parseOutcomeWithTimeout(0, func() parseSourceOutcome {
		return expected
	}, parseSourceOutcome{})

	if len(outcome.diagnostics) != 1 || outcome.diagnostics[0].Message != "ok" {
		t.Fatalf("unexpected outcome: %+v", outcome)
	}
}

func TestParseOutcomeWithTimeout_Timeout(t *testing.T) {
	timeoutOutcome := parseSourceOutcome{
		diagnostics: []Diagnostic{{
			Path:     "/tmp/Slow.kt",
			Message:  "parse timeout after 1ms",
			Severity: SeverityError,
		}},
	}

	outcome := parseOutcomeWithTimeout(1*time.Millisecond, func() parseSourceOutcome {
		time.Sleep(50 * time.Millisecond)
		return parseSourceOutcome{}
	}, timeoutOutcome)

	if len(outcome.diagnostics) != 1 {
		t.Fatalf("expected timeout diagnostics, got %+v", outcome)
	}
	if !strings.Contains(outcome.diagnostics[0].Message, "parse timeout after") {
		t.Fatalf("unexpected diagnostic message: %+v", outcome.diagnostics[0])
	}
}

func TestParseTimeoutOutcome(t *testing.T) {
	outcome := parseTimeoutOutcome("/tmp/Example.kt", 1250*time.Millisecond)

	if len(outcome.diagnostics) != 1 {
		t.Fatalf("expected one diagnostic, got %+v", outcome)
	}
	diag := outcome.diagnostics[0]
	if diag.Path != "/tmp/Example.kt" {
		t.Fatalf("unexpected path: %+v", diag)
	}
	if diag.Message != "parse timeout after 1.25s" {
		t.Fatalf("unexpected message: %+v", diag)
	}
}
