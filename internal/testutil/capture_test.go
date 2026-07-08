package testutil

import (
	"fmt"
	"os"
	"testing"
)

func TestCaptureOutput(t *testing.T) {
	code, stdout, stderr := CaptureOutput(t, func() int {
		fmt.Fprintln(os.Stdout, "out")
		fmt.Fprintln(os.Stderr, "err")
		return 7
	})

	if code != 7 {
		t.Fatalf("code mismatch: got=%d want=7", code)
	}
	if stdout != "out" {
		t.Fatalf("stdout mismatch: got=%q want=%q", stdout, "out")
	}
	if stderr != "err" {
		t.Fatalf("stderr mismatch: got=%q want=%q", stderr, "err")
	}
}
