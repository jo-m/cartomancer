//go:build online

package geocode

import (
	"archive/zip"
	"bufio"
	"os"
	"strings"
	"testing"

	"github.com/franiglesias/golden"
	"github.com/stretchr/testify/require"
	"jo-m.ch/go/detour/internal/pkg/utl"
)

// TestOnlineSubsampleAllCountries downloads allCountries.zip, extracts
// allCountries.txt, and writes every 1000th row to a golden file.
// The resulting testdata file is used by offline tests for parsing and
// database import without requiring a network download.
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
	lineNo := 0
	for scanner.Scan() {
		if lineNo%1000 == 0 {
			sampled = append(sampled, scanner.Text())
		}
		lineNo++
	}
	require.NoError(t, scanner.Err())
	require.NotEmpty(t, sampled)

	result := strings.Join(sampled, "\n") + "\n"
	golden.Verify(t, result, golden.Extension(".tsv")) // golden.WaitApproval()
}

// TestOnlineSubsampleAdmin1Codes downloads admin1CodesASCII.txt and writes
// every 10th row to a golden file for offline tests.
func TestOnlineSubsampleAdmin1Codes(t *testing.T) {
	data, err := utl.DownloadFile(Admin1CodesURL)
	require.NoError(t, err)
	require.NotEmpty(t, data)

	sampled := subsampleLines(string(data), 10)
	require.NotEmpty(t, sampled)

	golden.Verify(t, strings.Join(sampled, "\n")+"\n", golden.Extension(".tsv")) // golden.WaitApproval()
}

// TestOnlineSubsampleAdmin2Codes downloads admin2Codes.txt and writes
// every 10th row to a golden file for offline tests.
func TestOnlineSubsampleAdmin2Codes(t *testing.T) {
	data, err := utl.DownloadFile(Admin2CodesURL)
	require.NoError(t, err)
	require.NotEmpty(t, data)

	sampled := subsampleLines(string(data), 10)
	require.NotEmpty(t, sampled)

	golden.Verify(t, strings.Join(sampled, "\n")+"\n", golden.Extension(".tsv")) // golden.WaitApproval()
}

// subsampleLines returns every nth non-empty line from s.
func subsampleLines(s string, n int) []string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	var sampled []string
	for i, line := range lines {
		if i%n == 0 && line != "" {
			sampled = append(sampled, line)
		}
	}
	return sampled
}
