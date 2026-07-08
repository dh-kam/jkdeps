package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStartGraphProfilingNoPathsDoesNothing(t *testing.T) {
	originalStop := stopGraphProfiling
	t.Cleanup(func() {
		stopGraphProfiling = originalStop
	})
	t.Setenv("JKDEPS_CPU_PROFILE", "")
	t.Setenv("JKDEPS_MEM_PROFILE", "")

	if err := startGraphProfiling(); err != nil {
		t.Fatalf("startGraphProfiling() = %v, want nil", err)
	}
	stopGraphProfiling()
}

func TestStartGraphProfilingRejectsInvalidPath(t *testing.T) {
	originalStop := stopGraphProfiling
	t.Cleanup(func() {
		stopGraphProfiling = originalStop
	})
	t.Setenv("JKDEPS_CPU_PROFILE", filepath.Join(t.TempDir(), "missing", "dir", "jkdeps.cpuprofile"))
	if err := startGraphProfiling(); err == nil {
		t.Fatal("expected startGraphProfiling to fail on invalid cpu profile path")
	}
	stopGraphProfiling()
}

func TestStartAndStopGraphProfilingCreatesMemProfile(t *testing.T) {
	originalStop := stopGraphProfiling
	t.Cleanup(func() {
		stopGraphProfiling = originalStop
	})
	memProfilePath := filepath.Join(t.TempDir(), "jkdeps.memprofile")
	t.Setenv("JKDEPS_MEM_PROFILE", memProfilePath)
	t.Setenv("JKDEPS_CPU_PROFILE", "")

	if err := startGraphProfiling(); err != nil {
		t.Fatalf("startGraphProfiling() = %v", err)
	}
	stopGraphProfiling()

	if _, err := os.Stat(memProfilePath); err != nil {
		t.Fatalf("expected mem profile file: %v", err)
	}
}
