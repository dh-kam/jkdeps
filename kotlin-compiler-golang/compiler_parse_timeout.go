package kotlincompilergolang

import (
	"fmt"
	"time"
)

func (c *Compiler) parseWithConfiguredTimeout(path string, run func() parseSourceOutcome) parseSourceOutcome {
	return parseOutcomeWithTimeout(c.config.ParseTimeout, run, parseTimeoutOutcome(path, c.config.ParseTimeout))
}

func parseTimeoutOutcome(path string, timeout time.Duration) parseSourceOutcome {
	return parseSourceOutcome{
		tree: nil,
		diagnostics: []Diagnostic{{
			Path:     path,
			Line:     0,
			Column:   0,
			Message:  fmt.Sprintf("parse timeout after %s", timeout.Round(0)),
			Severity: SeverityError,
		}},
	}
}

func parseOutcomeWithTimeout(timeout time.Duration, run func() parseSourceOutcome, timeoutOutcome parseSourceOutcome) parseSourceOutcome {
	if timeout <= 0 {
		return run()
	}

	done := make(chan parseSourceOutcome, 1)
	go func() {
		done <- run()
	}()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case outcome := <-done:
		return outcome
	case <-timer.C:
		return timeoutOutcome
	}
}
