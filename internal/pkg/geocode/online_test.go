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
	"jo-m.ch/go/detour/internal/pkg/geocode/cols"
	"jo-m.ch/go/detour/internal/pkg/utl"
)

// TestOnlineGeoNamesLicense downloads the GeoNames readme and verifies that
// the license is still Creative Commons Attribution 4.0, matching DataAttribution.
func TestOnlineGeoNamesLicense(t *testing.T) {
	readmeURL := cols.BaseURL + "/readme.txt"
	data, err := utl.DownloadFile(t.Context(), readmeURL)
	require.NoError(t, err)

	readme := string(data)
	require.Contains(t, readme, "Creative Commons Attribution 4.0")
	require.Contains(t, readme, DataAttribution.LicenseURL)
}

// TestOnlineSubsampleAllCountries downloads allCountries.zip, extracts
// allCountries.txt, filters to Swiss (CH) entries only, and selects rows
// whose SHA-256 hash has 3 leading zero bits (~1/8 chance), producing a
// deterministic content-based subsample. The resulting testdata file is
// used by offline tests for parsing and database import without requiring
// a network download.
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
		if !isCountryLine(line, "CH") {
			continue
		}
		if hashSelectLine(line, 3) {
			sampled = append(sampled, line)
		}
	}
	require.NoError(t, scanner.Err())
	require.NotEmpty(t, sampled)

	result := strings.Join(sampled, "\n") + "\n"
	golden.Verify(t, result, golden.Extension(".tsv")) // golden.WaitApproval()
}

// TestOnlineSubsampleAdmin1Codes downloads admin1CodesASCII.txt and keeps
// all Swiss (CH.*) entries for offline tests.
func TestOnlineSubsampleAdmin1Codes(t *testing.T) {
	data, err := utl.DownloadFile(t.Context(), Admin1CodesURL)
	require.NoError(t, err)
	require.NotEmpty(t, data)

	sampled := filterLinesByPrefix(string(data), "CH.")
	require.NotEmpty(t, sampled)

	golden.Verify(t, strings.Join(sampled, "\n")+"\n", golden.Extension(".tsv")) // golden.WaitApproval()
}

// TestOnlineSubsampleAdmin2Codes downloads admin2Codes.txt and keeps
// all Swiss (CH.*) entries for offline tests.
func TestOnlineSubsampleAdmin2Codes(t *testing.T) {
	data, err := utl.DownloadFile(t.Context(), Admin2CodesURL)
	require.NoError(t, err)
	require.NotEmpty(t, data)

	sampled := filterLinesByPrefix(string(data), "CH.")
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

// isCountryLine returns true if the tab-delimited line has the given country
// code at the IdxCountryCode position.
func isCountryLine(line, countryCode string) bool {
	fields := strings.Split(line, "\t")
	if len(fields) <= cols.IdxCountryCode {
		return false
	}
	return fields[cols.IdxCountryCode] == countryCode
}

// filterLinesByPrefix returns non-empty lines from s that start with prefix.
func filterLinesByPrefix(s string, prefix string) []string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	var filtered []string
	for _, line := range lines {
		if line != "" && strings.HasPrefix(line, prefix) {
			filtered = append(filtered, line)
		}
	}
	return filtered
}
