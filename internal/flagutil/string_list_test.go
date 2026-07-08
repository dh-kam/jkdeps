package flagutil

import (
	"reflect"
	"testing"
)

func TestStringListFlagSetSplitsAndTrims(t *testing.T) {
	var got StringListFlag
	if err := got.Set(" a, b ,, c "); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	want := StringListFlag{"a", "b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Set() = %v, want %v", got, want)
	}
}

func TestStringListFlagSetIgnoresBlankInput(t *testing.T) {
	var got StringListFlag
	if err := got.Set("   "); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty flag value, got %v", got)
	}
}

func TestStringListNoSplitFlagPreservesCommas(t *testing.T) {
	var got StringListNoSplitFlag
	if err := got.Set(" a, b "); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	want := StringListNoSplitFlag{"a, b"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Set() = %v, want %v", got, want)
	}
}

func TestStringListNoSplitFlagSetIgnoresBlankInput(t *testing.T) {
	var got StringListNoSplitFlag
	if err := got.Set("   "); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty flag value, got %v", got)
	}
}

func TestUniqueStringsTrimsDeduplicatesAndPreservesOrder(t *testing.T) {
	in := []string{" a", "a", "", " b ", "a", "c", "  ", "b"}
	got := UniqueStrings(in)
	want := []string{"a", "b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("UniqueStrings() = %v, want %v", got, want)
	}
}

func TestStringListFlagString(t *testing.T) {
	f := StringListFlag{"a", "b", "c"}
	got := f.String()
	want := "a,b,c"
	if got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}

func TestStringListFlagStringEmpty(t *testing.T) {
	var f StringListFlag
	got := f.String()
	want := ""
	if got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}

func TestStringListNoSplitFlagString(t *testing.T) {
	f := StringListNoSplitFlag{"a, b", "c"}
	got := f.String()
	want := "a, b,c"
	if got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}

func TestStringListNoSplitFlagStringEmpty(t *testing.T) {
	var f StringListNoSplitFlag
	got := f.String()
	want := ""
	if got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}
