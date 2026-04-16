package geonames

import (
	"testing"
)

// BenchmarkParseLine benchmarks parsing a single geonames TSV line.
func BenchmarkParseLine(b *testing.B) {
	line := "2657896\tZurich\tZurich\tZurich,Zuerich\t47.36667\t8.55\tP\tPPLA\tCH\t\tZH\t112\t261\t\t415367\t408\t410\tEurope/Zurich\t2024-09-08"

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_, err := parseLine(line)
		if err != nil {
			b.Fatal(err)
		}
	}
}
