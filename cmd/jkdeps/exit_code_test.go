package main

import "testing"

func TestFailOnErrorExitCode(t *testing.T) {
	tests := []struct {
		name        string
		failOnError bool
		failedFiles int
		want        int
	}{
		{name: "disabled ignores failures", failOnError: false, failedFiles: 3, want: 0},
		{name: "enabled with no failures", failOnError: true, failedFiles: 0, want: 0},
		{name: "enabled with failures", failOnError: true, failedFiles: 1, want: 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := failOnErrorExitCode(tc.failOnError, tc.failedFiles); got != tc.want {
				t.Fatalf("failOnErrorExitCode(%v, %d) = %d, want %d", tc.failOnError, tc.failedFiles, got, tc.want)
			}
		})
	}
}
