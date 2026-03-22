//go:build online

package geoadmin_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/franiglesias/golden"
	"github.com/stretchr/testify/require"
	"jo-m.ch/go/detour/internal/pkg/geoadmin"
)

func toJSON(t *testing.T, val any) string {
	t.Helper()

	if val == nil {
		t.Fatal("toJSON: val is nil; cannot serialize a nil value")
	}

	ret, err := json.Marshal(val)
	if err != nil {
		t.Fatalf("toJSON: failed to marshal %T: %v", val, err)
	}

	res := string(ret)
	if res == "{}" || res == "null" {
		t.Fatalf("toJSON: produced an empty or null JSON (%s). Check if struct fields are exported.", res)
	}

	return res
}

func TestOnlineGetCatalog(t *testing.T) {
	ctx := context.Background()
	client := geoadmin.NewClient(geoadmin.BaseURL)

	catalog, err := client.GetCatalog(ctx)
	require.NoError(t, err)
	golden.Verify(t, toJSON(t, &catalog), golden.Extension(".json")) // golden.WaitApproval()
}

func TestOnlineGetCollections(t *testing.T) {
	ctx := context.Background()
	client := geoadmin.NewClient(geoadmin.BaseURL)

	collections, err := client.GetCollections(ctx, geoadmin.CollectionsParams{
		Limit:    3,
		Provider: "Agroscope",
	})
	require.NoError(t, err)
	require.Greater(t, len(collections), 3, "should have paginated through multiple pages")

	for _, c := range collections {
		require.NotEmpty(t, c.ID)
		require.NotEmpty(t, c.Description)
		require.Equal(t, "Collection", c.Type)
		require.NotEmpty(t, c.Links)
	}

	golden.Verify(t, toJSON(t, collections[:2]), golden.Extension(".json")) // golden.WaitApproval()
}

func TestOnlineGetCollection(t *testing.T) {
	ctx := context.Background()
	client := geoadmin.NewClient(geoadmin.BaseURL)

	col, err := client.GetCollection(ctx, "ch.swisstopo.swissalti3d")
	require.NoError(t, err)
	require.Equal(t, "ch.swisstopo.swissalti3d", col.ID)
	require.Equal(t, "Collection", col.Type)
	require.NotEmpty(t, col.Description)
	require.NotEmpty(t, col.Links)

	golden.Verify(t, toJSON(t, col), golden.Extension(".json")) // golden.WaitApproval()
}

func TestOnlineGetFeatures(t *testing.T) {
	ctx := context.Background()
	client := geoadmin.NewClient(geoadmin.BaseURL)

	items, err := client.GetFeatures(ctx, "ch.swisstopo.swissalti3d", geoadmin.FeaturesParams{
		Limit: 2,
		BBox:  [4]float64{7.43, 46.94, 7.44, 46.95},
	})
	require.NoError(t, err)
	require.Greater(t, len(items), 2, "should have paginated through multiple pages")

	for _, item := range items {
		require.NotEmpty(t, item.ID)
		require.Equal(t, "Feature", item.Type)
		require.NotEmpty(t, item.Links)
		require.False(t, item.Properties.Created.IsZero())
	}

	golden.Verify(t, toJSON(t, items[:2]), golden.Extension(".json")) // golden.WaitApproval()
}

func TestOnlineGetFeature(t *testing.T) {
	ctx := context.Background()
	client := geoadmin.NewClient(geoadmin.BaseURL)

	feature, err := client.GetFeature(ctx, "ch.swisstopo.swissalti3d", "swissalti3d_2019_2599-1198")
	require.NoError(t, err)
	require.Equal(t, "swissalti3d_2019_2599-1198", feature.ID)
	require.Equal(t, "Feature", feature.Type)
	require.Equal(t, "ch.swisstopo.swissalti3d", feature.Collection)
	require.NotNil(t, feature.Geometry)
	require.NotEmpty(t, feature.BBox)
	require.False(t, feature.Properties.Created.IsZero())
	require.NotEmpty(t, feature.Assets)
	require.NotEmpty(t, feature.Links)

	golden.Verify(t, toJSON(t, feature), golden.Extension(".json")) // golden.WaitApproval()
}

func TestOnlineGetAssets(t *testing.T) {
	ctx := context.Background()
	client := geoadmin.NewClient(geoadmin.BaseURL)

	assets, err := client.GetAssets(ctx, "ch.agroscope.abschaetzung-organische_boeden")
	require.NoError(t, err)
	require.NotEmpty(t, assets)

	for _, a := range assets {
		require.NotEmpty(t, a.ID)
		require.NotEmpty(t, a.Type)
		require.False(t, a.Created.IsZero())
		require.NotEmpty(t, a.Links)
	}

	golden.Verify(t, toJSON(t, assets), golden.Extension(".json")) // golden.WaitApproval()
}

func TestOnlineGetAsset(t *testing.T) {
	ctx := context.Background()
	client := geoadmin.NewClient(geoadmin.BaseURL)

	asset, err := client.GetAsset(ctx, "ch.agroscope.abschaetzung-organische_boeden", "abschaetzung-organische_boeden.zip")
	require.NoError(t, err)
	require.Equal(t, "abschaetzung-organische_boeden.zip", asset.ID)
	require.NotEmpty(t, asset.Type)
	require.NotNil(t, asset.Href)
	require.False(t, asset.Created.IsZero())
	require.False(t, asset.Updated.IsZero())
	require.NotEmpty(t, asset.Links)

	golden.Verify(t, toJSON(t, asset), golden.Extension(".json")) // golden.WaitApproval()
}

func TestOnlineDownloadAsset(t *testing.T) {
	ctx := context.Background()
	client := geoadmin.NewClient(geoadmin.BaseURL)

	asset, err := client.GetAsset(ctx, "ch.agroscope.abschaetzung-organische_boeden", "abschaetzung-organische_boeden.zip")
	require.NoError(t, err)

	body, contentType, err := geoadmin.DownloadAsset(ctx, *asset)
	require.NoError(t, err)
	defer body.Close()

	require.NotEmpty(t, contentType)

	// Read just the first few bytes to confirm the body is valid.
	buf := make([]byte, 4)
	n, err := body.Read(buf)
	require.NoError(t, err)
	require.Equal(t, 4, n)
	// ZIP magic bytes: PK\x03\x04.
	require.Equal(t, []byte{0x50, 0x4b, 0x03, 0x04}, buf)
}

func TestOnlineSearch(t *testing.T) {
	ctx := context.Background()
	client := geoadmin.NewClient(geoadmin.BaseURL)

	features, err := client.Search(ctx, geoadmin.SearchParams{
		Limit:       2,
		Collections: []string{"ch.swisstopo.swissalti3d"},
		BBox:        [4]float64{7.43, 46.94, 7.44, 46.95},
	})
	require.NoError(t, err)
	require.Greater(t, len(features), 2, "should have paginated through multiple pages")

	for _, f := range features {
		require.NotEmpty(t, f.ID)
		require.Equal(t, "Feature", f.Type)
		require.NotEmpty(t, f.Links)
	}

	golden.Verify(t, toJSON(t, features[:2]), golden.Extension(".json")) // golden.WaitApproval()
}

func TestOnlineSearchPost(t *testing.T) {
	ctx := context.Background()
	client := geoadmin.NewClient(geoadmin.BaseURL)

	bbox := [4]float64{7.43, 46.94, 7.44, 46.95}
	features, err := client.SearchPost(ctx, geoadmin.SearchPostBody{
		Collections: []string{"ch.swisstopo.swissalti3d"},
		Limit:       2,
		BBox:        &bbox,
	})
	require.NoError(t, err)
	require.Greater(t, len(features), 2, "should have paginated through multiple pages")

	for _, f := range features {
		require.NotEmpty(t, f.ID)
		require.Equal(t, "Feature", f.Type)
		require.NotEmpty(t, f.Links)
	}

	golden.Verify(t, toJSON(t, features[:2]), golden.Extension(".json")) // golden.WaitApproval()
}

// TODO: This will be out of date.
func TestOnlineGetForecastFeature(t *testing.T) {
	ctx := context.Background()
	client := geoadmin.NewClient(geoadmin.BaseURL)

	feature, err := client.GetFeature(ctx, "ch.meteoschweiz.ogd-forecasting-icon-ch1", "03212026-2100-0-aswdifu_s-perturbed-jku5bjo3")
	require.NoError(t, err)

	golden.Verify(t, toJSON(t, feature), golden.Extension(".json")) // golden.WaitApproval()
}
