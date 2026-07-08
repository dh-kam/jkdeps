package antlr

import (
	"testing"
	"unsafe"
)

var benchmarkPredictionContextSink *PredictionContext

func TestSingletonBasePredictionContextCreateReturnsSharedEmpty(t *testing.T) {
	got := SingletonBasePredictionContextCreate(nil, BasePredictionContextEmptyReturnState)
	if got != BasePredictionContextEMPTY {
		t.Fatalf("expected shared empty prediction context")
	}
}

func TestNewBaseSingletonPredictionContextInitializesFields(t *testing.T) {
	parent := NewBaseSingletonPredictionContext(BasePredictionContextEMPTY, 17)

	got := NewBaseSingletonPredictionContext(parent, 23)

	if got.pcType != PredictionContextSingleton {
		t.Fatalf("pcType = %d, want singleton", got.pcType)
	}
	if got.parentCtx != parent {
		t.Fatalf("parent pointer mismatch")
	}
	if got.returnState != 23 {
		t.Fatalf("returnState = %d, want 23", got.returnState)
	}
	if got.cachedHash == 0 {
		t.Fatalf("cachedHash should be initialized")
	}
}

func TestNewArrayPredictionContextInitializesFields(t *testing.T) {
	parentA := NewBaseSingletonPredictionContext(BasePredictionContextEMPTY, 3)
	parentB := NewBaseSingletonPredictionContext(BasePredictionContextEMPTY, 5)

	got := NewArrayPredictionContext([]*PredictionContext{parentA, parentB}, []int{7, 11})

	if got.pcType != PredictionContextArray {
		t.Fatalf("pcType = %d, want array", got.pcType)
	}
	if len(got.parents) != 2 || got.parents[0] != parentA || got.parents[1] != parentB {
		t.Fatalf("parents not preserved")
	}
	if len(got.returnStates) != 2 || got.returnStates[0] != 7 || got.returnStates[1] != 11 {
		t.Fatalf("returnStates not preserved")
	}
	if got.cachedHash == 0 {
		t.Fatalf("cachedHash should be initialized")
	}
}

func BenchmarkNewBaseSingletonPredictionContext(b *testing.B) {
	parent := NewBaseSingletonPredictionContext(BasePredictionContextEMPTY, 31)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchmarkPredictionContextSink = NewBaseSingletonPredictionContext(parent, 37)
	}
}

func BenchmarkNewArrayPredictionContext(b *testing.B) {
	parents := []*PredictionContext{
		NewBaseSingletonPredictionContext(BasePredictionContextEMPTY, 3),
		NewBaseSingletonPredictionContext(BasePredictionContextEMPTY, 5),
	}
	returnStates := []int{7, 11}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchmarkPredictionContextSink = NewArrayPredictionContext(parents, returnStates)
	}
}

func BenchmarkPredictionContextSize(b *testing.B) {
	b.ReportMetric(float64(unsafe.Sizeof(PredictionContext{})), "bytes/context")
}
