package antlr

import "testing"

func newBenchmarkPredictionContext(id int) *PredictionContext {
	parent := SingletonBasePredictionContextCreate(BasePredictionContextEMPTY, id+1)
	return SingletonBasePredictionContextCreate(parent, id+11)
}

func TestJPCMapPutAndGet(t *testing.T) {
	m := NewJPCMap(PredictionContextCacheCollection, "test")
	k1 := newBenchmarkPredictionContext(1)
	k2 := newBenchmarkPredictionContext(2)
	v := newBenchmarkPredictionContext(3)

	m.Put(k1, k2, v)

	got, ok := m.Get(k1, k2)
	if !ok {
		t.Fatalf("expected cached entry")
	}
	if got != v {
		t.Fatalf("got %p, want %p", got, v)
	}
}

func TestJPCMapDuplicatePutDoesNotReplace(t *testing.T) {
	m := NewJPCMap(PredictionContextCacheCollection, "test")
	k1 := newBenchmarkPredictionContext(10)
	k2 := newBenchmarkPredictionContext(20)
	first := newBenchmarkPredictionContext(30)
	second := newBenchmarkPredictionContext(40)

	m.Put(k1, k2, first)
	m.Put(k1, k2, second)

	got, ok := m.Get(k1, k2)
	if !ok {
		t.Fatalf("expected cached entry")
	}
	if got != first {
		t.Fatalf("duplicate Put replaced value: got %p want %p", got, first)
	}
}

func BenchmarkJPCMapPutDistinctPairs(b *testing.B) {
	keys1 := make([]*PredictionContext, b.N)
	keys2 := make([]*PredictionContext, b.N)
	vals := make([]*PredictionContext, b.N)
	for i := 0; i < b.N; i++ {
		keys1[i] = newBenchmarkPredictionContext(i * 3)
		keys2[i] = newBenchmarkPredictionContext(i*3 + 1)
		vals[i] = newBenchmarkPredictionContext(i*3 + 2)
	}

	m := NewJPCMap(PredictionContextCacheCollection, "bench")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Put(keys1[i], keys2[i], vals[i])
	}
}

func BenchmarkJPCMapGetHit(b *testing.B) {
	m := NewJPCMap(PredictionContextCacheCollection, "bench")
	k1 := newBenchmarkPredictionContext(101)
	k2 := newBenchmarkPredictionContext(202)
	v := newBenchmarkPredictionContext(303)
	m.Put(k1, k2, v)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		got, ok := m.Get(k1, k2)
		if !ok || got != v {
			b.Fatalf("cache miss during benchmark")
		}
	}
}
