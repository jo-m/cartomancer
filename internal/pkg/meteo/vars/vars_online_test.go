//go:build online

package vars

import (
	"context"
	"testing"

	"github.com/franiglesias/golden"
	"github.com/stretchr/testify/require"
	"jo-m.ch/go/cartomancer/internal/pkg/geoadmin"
	"jo-m.ch/go/cartomancer/internal/pkg/meteo/collection"
	"jo-m.ch/go/cartomancer/internal/pkg/utl"
)

// TestOnlineFetchVariablesCSV verifies that the variables CSV file is up to date.
func TestOnlineFetchVariablesCSV(t *testing.T) {
	ctx := context.Background()
	client := geoadmin.NewClient(geoadmin.BaseURL)
	coll, err := client.GetCollection(ctx, collection.ID)
	require.NoError(t, err)

	csvAsset, ok := coll.Assets[csvAssetKey]
	require.True(t, ok, "asset not found in collection")
	require.NotNil(t, csvAsset.Href, "asset has no href")

	csv, err := utl.DownloadFile(ctx, *csvAsset.Href)
	require.NoError(t, err)

	golden.Verify(t, string(csv), golden.Extension(".csv")) // golden.WaitApproval()
}
