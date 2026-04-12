// Package memstats provides helpers for reading and logging Go runtime memory statistics.
package memstats

import (
	"context"
	"runtime"
	"time"

	"jo-m.ch/go/cartomancer/internal/pkg/logg"
)

// Stats holds a point-in-time snapshot of selected memory statistics.
type Stats struct {
	// HeapAlloc is bytes of allocated heap objects.
	HeapAlloc uint64
	// HeapInuse is bytes in in-use heap spans.
	HeapInuse uint64
	// Sys is the total bytes of memory obtained from the OS.
	Sys uint64
	// TotalAlloc is cumulative bytes allocated (does not decrease).
	TotalAlloc uint64
	// NumGC is the number of completed GC cycles.
	NumGC uint32
}

// Snapshot reads the current memory statistics from the Go runtime.
func Snapshot() Stats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return Stats{
		HeapAlloc:  m.HeapAlloc,
		HeapInuse:  m.HeapInuse,
		Sys:        m.Sys,
		TotalAlloc: m.TotalAlloc,
		NumGC:      m.NumGC,
	}
}

// LogPeriodically logs memory statistics at the given interval until the context is cancelled.
// Intended to be called as a goroutine.
func LogPeriodically(ctx context.Context, interval time.Duration) {
	tick := time.NewTicker(interval)
	defer tick.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			s := Snapshot()
			logg.Debug(ctx, "memory stats",
				"heapAllocMB", s.HeapAlloc/(1024*1024),
				"heapInUseMB", s.HeapInuse/(1024*1024),
				"sysMB", s.Sys/(1024*1024),
				"numGC", s.NumGC,
			)
		}
	}
}
