package main

import (
	"os"
	"runtime"
	"runtime/pprof"
	"strings"
)

var stopCLIProfiling = func() {}
var stopGraphProfiling = func() {}

func startCLIProfiling() error {
	cpuProfilePath := strings.TrimSpace(os.Getenv("JKDEPS_CPU_PROFILE"))
	memProfilePath := strings.TrimSpace(os.Getenv("JKDEPS_MEM_PROFILE"))

	if cpuProfilePath == "" && memProfilePath == "" {
		return nil
	}

	stopCLIProfiling = func() {}

	if cpuProfilePath != "" {
		cpuFile, err := os.Create(cpuProfilePath)
		if err != nil {
			return err
		}
		if err := pprof.StartCPUProfile(cpuFile); err != nil {
			_ = cpuFile.Close()
			return err
		}
		oldStop := stopCLIProfiling
		stopCLIProfiling = func() {
			oldStop()
			pprof.StopCPUProfile()
			_ = cpuFile.Close()
		}
	}

	if memProfilePath != "" {
		memFile, err := os.Create(memProfilePath)
		if err != nil {
			stopCLIProfiling()
			stopCLIProfiling = func() {}
			return err
		}
		oldStop := stopCLIProfiling
		stopCLIProfiling = func() {
			oldStop()
			runtime.GC()
			_ = pprof.WriteHeapProfile(memFile)
			_ = memFile.Close()
		}
	}

	return nil
}

func startGraphProfiling() error {
	if err := startCLIProfiling(); err != nil {
		return err
	}
	stopGraphProfiling = func() {
		stopCLIProfiling()
	}
	return nil
}
