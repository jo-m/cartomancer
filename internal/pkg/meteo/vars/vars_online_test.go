//go:build online

package vars

import (
	"context"
	"testing"

	"github.com/franiglesias/golden"
	"github.com/stretchr/testify/require"
	"jo-m.ch/go/detour/internal/pkg/meteo/stac"
	"jo-m.ch/go/detour/internal/pkg/utl"
)

// TestOnlineFetchVariablesCSV verifies that the variables CSV file is up to date.
func TestOnlineFetchVariablesCSV(t *testing.T) {
	ctx := context.Background()
	coll, err := stac.FetchJSON[stac.Collection](ctx, stac.GetCollectionURL())
	require.NoError(t, err)

	csvAsset, ok := coll.Assets[CsvAssetKey]
	require.True(t, ok, "asset not found in collection")

	csv, err := utl.DownloadFile(csvAsset.Href)
	require.NoError(t, err)

	golden.Verify(t, string(csv), golden.Extension(".csv")) // golden.WaitApproval()
}
