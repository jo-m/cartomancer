//go:build online

package geocode

import (
	"archive/zip"
	"bufio"
	"crypto/sha256"
	"os"
	"strings"
	"testing"

	"github.com/franiglesias/golden"
	"github.com/stretchr/testify/require"
	"jo-m.ch/go/detour/internal/pkg/utl"
)

// TestOnlineSubsampleAllCountries downloads allCountries.zip, extracts
// allCountries.txt, and selects rows whose SHA-256 hash has 11 leading
// zero bits (~1/2048 chance), producing a deterministic content-based
// subsample. The resulting testdata file is used by offline tests for
// parsing and database import without requiring a network download.
func TestOnlineSubsampleAllCountries(t *testing.T) {
	ctx := t.Context()

	zipPath, err := DownloadAllCountries(ctx)
	require.NoError(t, err)
	defer os.Remove(zipPath)

	zr, err := zip.OpenReader(zipPath)
	require.NoError(t, err)
	defer zr.Close()

	var dataFile *zip.File
	for _, f := range zr.File {
		if f.Name == allCountriesEntry {
			dataFile = f
			break
		}
	}
	require.NotNil(t, dataFile, "allCountries.txt not found in zip")

	rc, err := dataFile.Open()
	require.NoError(t, err)
	defer rc.Close()

	scanner := bufio.NewScanner(rc)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	var sampled []string
	for scanner.Scan() {
		line := scanner.Text()
		if hashSelectLine(line, 11) {
			sampled = append(sampled, line)
		}
	}
	require.NoError(t, scanner.Err())
	require.NotEmpty(t, sampled)

	result := strings.Join(sampled, "\n") + "\n"
	golden.Verify(t, result, golden.Extension(".tsv")) // golden.WaitApproval()
}

// TestOnlineSubsampleAdmin1Codes downloads admin1CodesASCII.txt and writes
// every ~16th row (4 leading zero bits) to a golden file for offline tests.
func TestOnlineSubsampleAdmin1Codes(t *testing.T) {
	data, err := utl.DownloadFile(Admin1CodesURL)
	require.NoError(t, err)
	require.NotEmpty(t, data)

	sampled := subsampleLines(string(data), 4)
	require.NotEmpty(t, sampled)

	golden.Verify(t, strings.Join(sampled, "\n")+"\n", golden.Extension(".tsv")) // golden.WaitApproval()
}

// TestOnlineSubsampleAdmin2Codes downloads admin2Codes.txt and writes
// every ~16th row (4 leading zero bits) to a golden file for offline tests.
func TestOnlineSubsampleAdmin2Codes(t *testing.T) {
	data, err := utl.DownloadFile(Admin2CodesURL)
	require.NoError(t, err)
	require.NotEmpty(t, data)

	sampled := subsampleLines(string(data), 4)
	require.NotEmpty(t, sampled)

	golden.Verify(t, strings.Join(sampled, "\n")+"\n", golden.Extension(".tsv")) // golden.WaitApproval()
}

// hashSelectLine returns true if the SHA-256 hash of line has at least
// nZeroBits leading zero bits, providing a deterministic, content-based
// subsample that is stable across row reordering.
func hashSelectLine(line string, nZeroBits int) bool {
	h := sha256.Sum256([]byte(line))
	for i := range nZeroBits {
		byteIdx := i / 8
		bitIdx := 7 - (i % 8)
		if h[byteIdx]&(1<<bitIdx) != 0 {
			return false
		}
	}
	return true
}

// subsampleLines returns non-empty lines from s whose SHA-256 hash has at
// least nZeroBits leading zero bits.
func subsampleLines(s string, nZeroBits int) []string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	var sampled []string
	for _, line := range lines {
		if line != "" && hashSelectLine(line, nZeroBits) {
			sampled = append(sampled, line)
		}
	}
	return sampled
}
