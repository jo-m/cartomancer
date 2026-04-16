//go:build online

package geonames

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"jo-m.ch/go/cartomancer/internal/pkg/db/geonamesdb"
	"jo-m.ch/go/cartomancer/internal/pkg/logg"
)

const benchMaxRows = 1_000_000

// peakTracker polls runtime.MemStats and /proc/self/status at a fixed interval,
// recording the high-water marks for heap-in-use, sys, and OS-reported RSS.
type peakTracker struct {
	interval     time.Duration
	stop         atomic.Bool
	peakHeap     uint64
	peakSys      uint64
	peakRSSBytes uint64
}

// start begins background polling. Call [peakTracker.finish] to stop and read results.
func (p *peakTracker) start() {
	p.interval = 50 * time.Millisecond
	go func() {
		var ms runtime.MemStats
		for !p.stop.Load() {
			runtime.ReadMemStats(&ms)
			if ms.HeapInuse > p.peakHeap {
				p.peakHeap = ms.HeapInuse
			}
			if ms.Sys > p.peakSys {
				p.peakSys = ms.Sys
			}
			if rss := readVmRSS(); rss > p.peakRSSBytes {
				p.peakRSSBytes = rss
			}
			time.Sleep(p.interval)
		}
	}()
}

// finish stops polling.
func (p *peakTracker) finish() {
	p.stop.Store(true)
}

// readVmRSS reads VmRSS from /proc/self/status (Linux only).
// Returns 0 on any error.
func readVmRSS() uint64 {
	data, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return 0
	}
	var value uint64
	// VmRSS is reported in kB.
	_, err = fmt.Sscanf(findLine(data, "VmRSS:"), "VmRSS: %d kB", &value)
	if err != nil {
		return 0
	}
	return value * 1024
}

// findLine returns the first line in data starting with prefix.
func findLine(data []byte, prefix string) string {
	for i := 0; i < len(data); {
		end := i
		for end < len(data) && data[end] != '\n' {
			end++
		}
		line := string(data[i:end])
		if len(line) >= len(prefix) && line[:len(prefix)] == prefix {
			return line
		}
		i = end + 1
	}
	return ""
}

// BenchmarkOnlineDownloadAndImport benchmarks the production import pipeline
// with up to 1M rows from the real allCountries dataset. Downloads are
// performed once before the timed section.
//
// Run with:
//
//	go test -tags online -run='^$' -bench=BenchmarkOnlineDownloadAndImport -benchmem -benchtime=1x ./internal/pkg/geonames/
func BenchmarkOnlineDownloadAndImport(b *testing.B) {
	ctx := logg.WithDiscardHandler(context.Background())

	// Download all data before the benchmark timer starts.
	zipPath, err := DownloadAllCountries(ctx)
	require.NoError(b, err)
	defer os.Remove(zipPath)

	admin1Data, admin2Data, err := DownloadAdminCodes(ctx)
	require.NoError(b, err)

	// Open the zip and find allCountries.txt once.
	zr, err := zip.OpenReader(zipPath)
	require.NoError(b, err)
	defer zr.Close()

	var dataFile *zip.File
	for _, f := range zr.File {
		if f.Name == allCountriesEntry {
			dataFile = f
			break
		}
	}
	require.NotNil(b, dataFile, "allCountries.txt not found in zip")

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		dir := b.TempDir()
		d, err := geonamesdb.Open(ctx, filepath.Join(dir, "geonames.db"))
		require.NoError(b, err)

		runtime.GC()

		var pt peakTracker
		pt.start()

		// Import allCountries (capped to benchMaxRows).
		rc, err := dataFile.Open()
		require.NoError(b, err)

		rowCount, err := importFromReader(ctx, d, rc, benchMaxRows)
		rc.Close()
		require.NoError(b, err)

		// Import admin codes (same as production, small datasets).
		_, err = ImportAdmin1Codes(ctx, d, bytes.NewReader(admin1Data))
		require.NoError(b, err)
		_, err = ImportAdmin2Codes(ctx, d, bytes.NewReader(admin2Data))
		require.NoError(b, err)

		pt.finish()

		b.ReportMetric(float64(rowCount), "rows")
		b.ReportMetric(float64(pt.peakHeap), "peak-heap-bytes")
		b.ReportMetric(float64(pt.peakSys), "peak-sys-bytes")
		b.ReportMetric(float64(pt.peakRSSBytes), "peak-rss-bytes")

		d.Close()
	}
}
