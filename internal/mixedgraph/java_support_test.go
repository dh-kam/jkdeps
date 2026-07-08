package mixedgraph

import "testing"

func TestNormalizePathTrimsBackticksAndWhitespace(t *testing.T) {
	got := normalizePath("  `com`.example .  ")
	if got != "com.example" {
		t.Fatalf("expected com.example, got %q", got)
	}
}

func TestNormalizePathWithoutCleanupNoAllocationLikePath(t *testing.T) {
	got := normalizePath("com.example.package")
	if got != "com.example.package" {
		t.Fatalf("expected input to stay unchanged, got %q", got)
	}
}

func TestNormalizePathTrimsDotsAndOnlyIgnoredCharacters(t *testing.T) {
	got := normalizePath(" ..`com.example.Type`.. ")
	if got != "com.example.Type" {
		t.Fatalf("expected com.example.Type, got %q", got)
	}

	got = normalizePath(" .. ` \t ")
	if got != "" {
		t.Fatalf("expected empty string after trimming ignored characters, got %q", got)
	}
}

func TestInferImportPackageKeepsKnownBehavior(t *testing.T) {
	got := inferImportPackage("com.example.util.String")
	if got != "com.example.util" {
		t.Fatalf("expected com.example.util, got %q", got)
	}

	got = inferImportPackage("com.example.*")
	if got != "com.example" {
		t.Fatalf("expected wildcard to trim, got %q", got)
	}
}
