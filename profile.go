package main

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
)

type applicationProfiler struct {
	cpuFile        *os.File
	memProfilePath string
}

func startApplicationProfiler(cpuProfilePath, memProfilePath string) (*applicationProfiler, error) {
	profiler := &applicationProfiler{memProfilePath: memProfilePath}
	if cpuProfilePath == "" {
		return profiler, nil
	}

	cpuFile, err := os.Create(cpuProfilePath)
	if err != nil {
		return nil, fmt.Errorf("create CPU profile: %w", err)
	}
	if err := pprof.StartCPUProfile(cpuFile); err != nil {
		_ = cpuFile.Close()
		return nil, fmt.Errorf("start CPU profile: %w", err)
	}
	profiler.cpuFile = cpuFile

	return profiler, nil
}

func (p *applicationProfiler) Stop() error {
	var profileErrors []error
	if p.cpuFile != nil {
		pprof.StopCPUProfile()
		if err := p.cpuFile.Close(); err != nil {
			profileErrors = append(profileErrors, fmt.Errorf("close CPU profile: %w", err))
		}
	}

	if p.memProfilePath != "" {
		if err := writeHeapProfile(p.memProfilePath); err != nil {
			profileErrors = append(profileErrors, err)
		}
	}

	return errors.Join(profileErrors...)
}

func writeHeapProfile(path string) error {
	memFile, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create memory profile: %w", err)
	}

	runtime.GC()
	writeErr := pprof.WriteHeapProfile(memFile)
	closeErr := memFile.Close()
	if writeErr != nil {
		return fmt.Errorf("write memory profile: %w", writeErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close memory profile: %w", closeErr)
	}

	return nil
}
