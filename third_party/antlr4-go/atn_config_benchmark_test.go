package antlr

import (
	"testing"
	"unsafe"
)

var benchmarkATNConfigSink *ATNConfig

func newBenchmarkATNConfig(stateNumber int) *ATNConfig {
	state := NewBasicState()
	state.SetStateNumber(stateNumber)
	return NewATNConfig5(state, 7, SingletonBasePredictionContextCreate(BasePredictionContextEMPTY, 13), SemanticContextNone)
}

func TestNewATNConfig4CopiesParserFields(t *testing.T) {
	source := newBenchmarkATNConfig(11)
	source.SetReachesIntoOuterContext(5)
	source.setPrecedenceFilterSuppressed(true)

	targetState := NewBasicState()
	targetState.SetStateNumber(17)

	got := NewATNConfig4(source, targetState)

	if got.GetState().GetStateNumber() != 17 {
		t.Fatalf("state = %d, want 17", got.GetState().GetStateNumber())
	}
	if got.GetAlt() != source.GetAlt() {
		t.Fatalf("alt = %d, want %d", got.GetAlt(), source.GetAlt())
	}
	if got.GetContext() != source.GetContext() {
		t.Fatalf("context pointer mismatch")
	}
	if got.GetSemanticContext() != source.GetSemanticContext() {
		t.Fatalf("semantic context mismatch")
	}
	if got.GetReachesIntoOuterContext() != source.GetReachesIntoOuterContext() {
		t.Fatalf("reachesIntoOuterContext = %d, want %d", got.GetReachesIntoOuterContext(), source.GetReachesIntoOuterContext())
	}
	if !got.getPrecedenceFilterSuppressed() {
		t.Fatalf("precedenceFilterSuppressed should be copied")
	}
	if got.cType != parserConfig {
		t.Fatalf("cType = %d, want parserConfig", got.cType)
	}
}

func TestNewATNConfig5InitializesParserFields(t *testing.T) {
	state := NewBasicState()
	state.SetStateNumber(23)
	ctx := SingletonBasePredictionContextCreate(BasePredictionContextEMPTY, 19)

	got := NewATNConfig5(state, 3, ctx, SemanticContextNone)

	if got.GetState().GetStateNumber() != 23 {
		t.Fatalf("state = %d, want 23", got.GetState().GetStateNumber())
	}
	if got.GetAlt() != 3 {
		t.Fatalf("alt = %d, want 3", got.GetAlt())
	}
	if got.GetContext() != ctx {
		t.Fatalf("context pointer mismatch")
	}
	if got.GetSemanticContext() != SemanticContextNone {
		t.Fatalf("semantic context mismatch")
	}
	if got.cType != parserConfig {
		t.Fatalf("cType = %d, want parserConfig", got.cType)
	}
}

func BenchmarkNewATNConfig4(b *testing.B) {
	source := newBenchmarkATNConfig(31)
	targetState := NewBasicState()
	targetState.SetStateNumber(37)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchmarkATNConfigSink = NewATNConfig4(source, targetState)
	}
}

func BenchmarkNewATNConfig5(b *testing.B) {
	state := NewBasicState()
	state.SetStateNumber(41)
	ctx := SingletonBasePredictionContextCreate(BasePredictionContextEMPTY, 29)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchmarkATNConfigSink = NewATNConfig5(state, 9, ctx, SemanticContextNone)
	}
}

func BenchmarkATNConfigSize(b *testing.B) {
	b.ReportMetric(float64(unsafe.Sizeof(ATNConfig{})), "bytes/config")
}
