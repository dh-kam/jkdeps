// Package ast provides profiling support for parser performance analysis
package ast

import (
	"os"
	"runtime"
	"runtime/pprof"
	"testing"
	"time"
)

// ProfilingOptions controls profiling behavior
type ProfilingOptions struct {
	// CPUProfile enables CPU profiling
	CPUProfile bool
	// MemProfile enables memory profiling
	MemProfile bool
	// BlockProfile enables block (goroutine contention) profiling
	BlockProfile bool
	// CPUProfileFile is the file path for CPU profile output
	CPUProfileFile string
	// MemProfileFile is the file path for memory profile output
	MemProfileFile string
	// BlockProfileFile is the file path for block profile output
	BlockProfileFile string
}

// StartProfiling starts profiling based on the given options
// Returns a stop function that must be called to end profiling
func StartProfiling(opts ProfilingOptions) (stop func(), err error) {
	var cleaners []func()
	var stopFuncs []func()

	cleanup := func() {
		for i := len(cleaners) - 1; i >= 0; i-- {
			cleaners[i]()
		}
	}

	defer func() {
		if err != nil {
			cleanup()
		}
	}()

	if opts.CPUProfile {
		f, err := os.Create(opts.CPUProfileFile)
		if err != nil {
			return nil, err
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			f.Close()
			return nil, err
		}
		cleaners = append(cleaners, func() {
			pprof.StopCPUProfile()
			f.Close()
		})
	}

	if opts.BlockProfile {
		runtime.SetBlockProfileRate(1)
		cleaners = append(cleaners, func() {
			runtime.SetBlockProfileRate(0)
		})
	}

	stopFuncs = append(stopFuncs, func() {
		if opts.MemProfile {
			f, err := os.Create(opts.MemProfileFile)
			if err == nil {
				runtime.GC() // Force GC before taking snapshot
				pprof.WriteHeapProfile(f)
				f.Close()
			}
		}
		if opts.BlockProfile {
			f, err := os.Create(opts.BlockProfileFile)
			if err == nil {
				pprof.Lookup("block").WriteTo(f, 0)
				f.Close()
			}
		}
	})

	return func() {
		for _, f := range stopFuncs {
			f()
		}
		cleanup()
	}, nil
}

// BenchmarkResult represents benchmark results for a parser
type BenchmarkResult struct {
	Name           string        `json:"name"`
	Iterations     int           `json:"iterations"`
	Duration       time.Duration `json:"duration"`
	AvgDuration    time.Duration `json:"avg_duration"`
	MinDuration    time.Duration `json:"min_duration"`
	MaxDuration    time.Duration `json:"max_duration"`
	BytesAllocated uint64        `json:"bytes_allocated"`
	AllocsPerRun   uint64        `json:"allocs_per_run"`
}

// BenchmarkParser runs a benchmark on the given parser function
func BenchmarkParser(b *testing.B, name string, parseFn func()) BenchmarkResult {
	var durations []time.Duration
	var m1, m2 runtime.MemStats

	runtime.GC()
	runtime.ReadMemStats(&m1)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		start := time.Now()
		parseFn()
		durations = append(durations, time.Since(start))
	}

	runtime.ReadMemStats(&m2)

	// Calculate statistics
	minDur := durations[0]
	maxDur := durations[0]
	totalDur := time.Duration(0)
	for _, d := range durations {
		totalDur += d
		if d < minDur {
			minDur = d
		}
		if d > maxDur {
			maxDur = d
		}
	}

	return BenchmarkResult{
		Name:           name,
		Iterations:     b.N,
		Duration:       totalDur,
		AvgDuration:    totalDur / time.Duration(b.N),
		MinDuration:    minDur,
		MaxDuration:    maxDur,
		BytesAllocated: m2.TotalAlloc - m1.TotalAlloc,
		AllocsPerRun:   (m2.TotalAlloc - m1.TotalAlloc) / uint64(b.N),
	}
}

// MemoryUsage captures current memory usage statistics
type MemoryUsage struct {
	Alloc       uint64 `json:"alloc"`        // Currently allocated memory
	TotalAlloc  uint64 `json:"total_alloc"`  // Cumulative allocated memory
	Sys         uint64 `json:"sys"`          // Total memory obtained from OS
	NumGC       uint32 `json:"num_gc"`       // Number of GC runs
	HeapAlloc   uint64 `json:"heap_alloc"`   // Heap allocation
	HeapSys     uint64 `json:"heap_sys"`     // Heap system memory
	HeapObjects uint64 `json:"heap_objects"` // Heap object count
	StackInuse  uint64 `json:"stack_inuse"`  // Stack in-use memory
	MSpanInuse  uint64 `json:"mspan_inuse"`  // MSpan in-use memory
	MCacheInuse uint64 `json:"mcache_inuse"` // MCache in-use memory
}

// GetMemoryUsage captures current memory usage
func GetMemoryUsage() MemoryUsage {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return MemoryUsage{
		Alloc:       m.Alloc,
		TotalAlloc:  m.TotalAlloc,
		Sys:         m.Sys,
		NumGC:       m.NumGC,
		HeapAlloc:   m.HeapAlloc,
		HeapSys:     m.HeapSys,
		HeapObjects: m.HeapObjects,
		StackInuse:  m.StackInuse,
		MSpanInuse:  m.MSpanInuse,
		MCacheInuse: m.MCacheInuse,
	}
}

// MemoryTracker tracks memory usage during parsing
type MemoryTracker struct {
	before MemoryUsage
	after  MemoryUsage
}

// StartMemoryTracking starts tracking memory usage
func StartMemoryTracking() *MemoryTracker {
	return &MemoryTracker{
		before: GetMemoryUsage(),
	}
}

// Stop stops tracking and returns memory usage delta
func (mt *MemoryTracker) Stop() MemoryUsage {
	mt.after = GetMemoryUsage()
	return mt.Delta()
}

// Before returns the memory usage before tracking started
func (mt *MemoryTracker) Before() MemoryUsage {
	return mt.before
}

// After returns the memory usage after tracking stopped
func (mt *MemoryTracker) After() MemoryUsage {
	return mt.after
}

// Delta returns the memory usage difference
func (mt *MemoryTracker) Delta() MemoryUsage {
	return MemoryUsage{
		Alloc:       mt.after.Alloc - mt.before.Alloc,
		TotalAlloc:  mt.after.TotalAlloc - mt.before.TotalAlloc,
		HeapAlloc:   mt.after.HeapAlloc - mt.before.HeapAlloc,
		HeapObjects: mt.after.HeapObjects - mt.before.HeapObjects,
	}
}

// ParseMetrics represents metrics collected during parsing
type ParseMetrics struct {
	FileName        string         `json:"file_name"`
	FileSize        int64          `json:"file_size"`
	Language        SourceLanguage `json:"language"`
	ParseDuration   time.Duration  `json:"parse_duration"`
	MemoryBefore    MemoryUsage    `json:"memory_before,omitempty"`
	MemoryAfter     MemoryUsage    `json:"memory_after,omitempty"`
	MemoryDelta     MemoryUsage    `json:"memory_delta,omitempty"`
	Success         bool           `json:"success"`
	DiagnosticCount int            `json:"diagnostic_count"`
}

// TimedParse runs parseFn and returns metrics
func TimedParse(fileName string, fileSize int64, lang SourceLanguage, parseFn func() error) ParseMetrics {
	metrics := ParseMetrics{
		FileName: fileName,
		FileSize: fileSize,
		Language: lang,
	}

	tracker := StartMemoryTracking()

	start := time.Now()
	err := parseFn()
	metrics.ParseDuration = time.Since(start)

	metrics.MemoryBefore = tracker.before
	metrics.MemoryAfter = GetMemoryUsage()
	metrics.MemoryDelta = tracker.Delta()

	metrics.Success = err == nil

	return metrics
}
