//go:build online

package params

import (
	"testing"

	"github.com/franiglesias/golden"
	"github.com/stretchr/testify/require"
	"jo-m.ch/go/detour/internal/pkg/utl"
)

const wmoCSVURL = "https://codes.wmo.int/grib2/codeflag/4.2?_format=csv&status=valid"

// TestOnlineFetchParamsCSV verifies that the params CSV file is up to date.
func TestOnlineFetchParamsCSV(t *testing.T) {
	csv, err := utl.DownloadFile(t.Context(), wmoCSVURL)
	require.NoError(t, err)

	golden.Verify(t, string(csv), golden.Extension(".csv")) // golden.WaitApproval()
}
