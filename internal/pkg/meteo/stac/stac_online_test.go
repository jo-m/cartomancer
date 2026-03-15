//go:build online

package stac_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"jo-m.ch/go/detour/internal/pkg/meteo/stac"
	"jo-m.ch/go/detour/internal/pkg/utl"
)

func TestOnlineStacLicense(t *testing.T) {
	ctx := context.Background()
	coll, err := stac.FetchJSON[stac.Collection](ctx, stac.GetCollectionURL())
	require.NoError(t, err)

	// Ensure data license has not changed.
	require.Equal(t, "CC-BY", coll.License)
}

func TestOnlineStacCRS(t *testing.T) {
	ctx := context.Background()
	coll, err := stac.FetchJSON[stac.Collection](ctx, stac.GetCollectionURL())
	require.NoError(t, err)

	// Ensure WGS84 is used.
	require.Len(t, coll.CRS, 1)
	require.Equal(t, "http://www.opengis.net/def/crs/OGC/1.3/CRS84", coll.CRS[0])
}

func TestOnlineStacBasicParse(t *testing.T) {
	ctx := context.Background()
	coll, err := stac.FetchJSON[stac.Collection](ctx, stac.GetCollectionURL())
	require.NoError(t, err)

	require.NotEmpty(t, coll.Assets)
}

func TestOnlineStacCollectionNewestReferenceTime(t *testing.T) {
	ctx := context.Background()
	coll, err := stac.FetchJSON[stac.Collection](ctx, stac.GetCollectionURL())
	require.NoError(t, err)

	refTime := coll.NewestReferenceTime()
	require.False(t, refTime.IsZero(), "expected non-zero reference time from collection extent")
	t.Logf("newest reference time: %s", refTime)
}

func TestOnlineStacSearchItems(t *testing.T) {
	ctx := context.Background()
	coll, err := stac.FetchJSON[stac.Collection](ctx, stac.GetCollectionURL())
	require.NoError(t, err)

	// The collection extent may advertise a model run that is still uploading,
	// so step back in 3 h increments until we find one with data.
	refTime := coll.NewestReferenceTime()
	t.Logf("initial refTime from extent: %s", refTime)
	require.False(t, refTime.IsZero())

	var items []stac.Item
	for attempt := range 8 {
		candidate := refTime.Add(-time.Duration(attempt) * 3 * time.Hour)
		items, err = stac.SearchItems(ctx, stac.SearchReq{
			Collections:         []string{stac.CollectionID},
			ForecastRefDatetime: candidate.Format(time.RFC3339),
			ForecastVariable:    "T_2M",
			ForecastPerturbed:   utl.Ptr(false),
		})
		require.NoError(t, err)
		if len(items) > 0 {
			refTime = candidate
			t.Logf("found data at refTime: %s (attempt %d)", refTime, attempt)
			break
		}
	}
	require.Len(t, items, 34)

	for _, item := range items {
		require.Equal(t, "T_2M", item.Props.Variable)
		require.Equal(t, refTime, item.Props.ReferenceDatetime)
		require.False(t, item.Props.Perturbed)
		t.Logf("item: %+v", item.Props)
	}
}

func TestOnlineStacFetchItemsForVariables(t *testing.T) {
	ctx := context.Background()
	items, coll, err := stac.FetchItemsForVariables(ctx, stac.GetCollectionURL(), []string{"T_2M", "U"}, false)
	require.NoError(t, err)
	require.NotNil(t, coll)
	require.Len(t, items, 34*2)

	refTime := items[0].Props.ReferenceDatetime
	t.Logf("refTime: %s", refTime)

	for _, item := range items {
		require.False(t, item.Props.Perturbed)
		require.Equal(t, refTime, item.Props.ReferenceDatetime)
		horz, err := stac.ParseISO8601Duration(item.Props.Horizon)
		require.NoError(t, err)
		require.Equal(t, item.Props.ReferenceDatetime.Add(horz), item.Props.ValidDatetime)
		t.Logf("item: %+v", item.Props)
	}
}
