package antlr

import (
	"strconv"
	"testing"
)

type benchmarkKey struct {
	hash int
	id   int
}

type benchmarkComparator struct{}

func (benchmarkComparator) Hash1(v benchmarkKey) int {
	return v.hash
}

func (benchmarkComparator) Equals2(a, b benchmarkKey) bool {
	return a.id == b.id
}

func TestJMapPutGetDeleteWithHashCollisions(t *testing.T) {
	m := NewJMap[benchmarkKey, string, benchmarkComparator](benchmarkComparator{}, UnknownCollection, "test")

	keys := []benchmarkKey{
		{hash: 7, id: 1},
		{hash: 7, id: 2},
		{hash: 7, id: 3},
	}
	for _, key := range keys {
		if _, exists := m.Put(key, "v"+strconv.Itoa(key.id)); exists {
			t.Fatalf("unexpected existing key on first insert: %+v", key)
		}
	}

	for _, key := range keys {
		got, ok := m.Get(key)
		if !ok {
			t.Fatalf("missing key after insert: %+v", key)
		}
		want := "v" + strconv.Itoa(key.id)
		if got != want {
			t.Fatalf("Get(%+v) = %q, want %q", key, got, want)
		}
	}

	if got, exists := m.Put(keys[1], "ignored"); !exists || got != "v2" {
		t.Fatalf("duplicate Put returned (%q, %v), want (%q, true)", got, exists, "v2")
	}

	m.Delete(keys[1])
	if _, ok := m.Get(keys[1]); ok {
		t.Fatalf("key should be deleted: %+v", keys[1])
	}
	if m.Len() != 2 {
		t.Fatalf("Len() = %d, want 2", m.Len())
	}
}

func TestJStorePutGetWithHashCollisions(t *testing.T) {
	s := NewJStore[benchmarkKey, benchmarkComparator](benchmarkComparator{}, UnknownCollection, "test")

	keys := []benchmarkKey{
		{hash: 11, id: 10},
		{hash: 11, id: 11},
		{hash: 11, id: 12},
	}
	for _, key := range keys {
		if _, exists := s.Put(key); exists {
			t.Fatalf("unexpected existing value on first insert: %+v", key)
		}
	}

	for _, key := range keys {
		got, ok := s.Get(key)
		if !ok {
			t.Fatalf("missing key after insert: %+v", key)
		}
		if got != key {
			t.Fatalf("Get(%+v) = %+v, want %+v", key, got, key)
		}
	}

	if got, exists := s.Put(keys[2]); !exists || got != keys[2] {
		t.Fatalf("duplicate Put returned (%+v, %v), want (%+v, true)", got, exists, keys[2])
	}
}

func BenchmarkJMapPut_Unique(b *testing.B) {
	keys := make([]benchmarkKey, b.N)
	for i := range keys {
		keys[i] = benchmarkKey{hash: i + 1, id: i + 1}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m := NewJMap[benchmarkKey, int, benchmarkComparator](benchmarkComparator{}, UnknownCollection, "bench")
		_, _ = m.Put(keys[i], i)
	}
}

func BenchmarkJMapPut_SharedBucket(b *testing.B) {
	keys := make([]benchmarkKey, b.N)
	for i := range keys {
		keys[i] = benchmarkKey{hash: 1, id: i + 1}
	}

	m := NewJMap[benchmarkKey, int, benchmarkComparator](benchmarkComparator{}, UnknownCollection, "bench")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = m.Put(keys[i], i)
	}
}

func BenchmarkJStorePut_SharedBucket(b *testing.B) {
	keys := make([]benchmarkKey, b.N)
	for i := range keys {
		keys[i] = benchmarkKey{hash: 1, id: i + 1}
	}

	s := NewJStore[benchmarkKey, benchmarkComparator](benchmarkComparator{}, UnknownCollection, "bench")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = s.Put(keys[i])
	}
}
