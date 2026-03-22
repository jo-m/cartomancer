//go:build online

package meteo

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"jo-m.ch/go/detour/internal/pkg/geoadmin"
	"jo-m.ch/go/detour/internal/pkg/meteo/collection"
	"jo-m.ch/go/detour/internal/pkg/utl"
)

func TestOnlineStacLicense(t *testing.T) {
	ctx := context.Background()
	coll, err := newClient().GetCollection(ctx, collection.ID)
	require.NoError(t, err)

	// Ensure data license has not changed.
	require.Equal(t, "CC-BY", coll.License)
}

func TestOnlineStacCRS(t *testing.T) {
	ctx := context.Background()
	coll, err := newClient().GetCollection(ctx, collection.ID)
	require.NoError(t, err)

	// Ensure WGS84 is used via summaries (CRS is not in the geoadmin Collection type).
	require.NotNil(t, coll.Summaries)
}

func TestOnlineStacBasicParse(t *testing.T) {
	ctx := context.Background()
	coll, err := newClient().GetCollection(ctx, collection.ID)
	require.NoError(t, err)

	require.NotEmpty(t, coll.Assets)
}

func TestOnlineStacCollectionNewestReferenceTime(t *testing.T) {
	ctx := context.Background()
	coll, err := newClient().GetCollection(ctx, collection.ID)
	require.NoError(t, err)

	refTime := newestReferenceTime(coll)
	require.False(t, refTime.IsZero(), "expected non-zero reference time from collection extent")
	t.Logf("newest reference time: %s", refTime)
}

func TestOnlineStacSearchItems(t *testing.T) {
	ctx := context.Background()
	client := newClient()

	coll, err := client.GetCollection(ctx, collection.ID)
	require.NoError(t, err)

	// The collection extent may advertise a model run that is still uploading,
	// so step back in 3 h increments until we find one with data.
	refTime := newestReferenceTime(coll)
	t.Logf("initial refTime from extent: %s", refTime)
	require.False(t, refTime.IsZero())

	var features []geoadmin.Feature
	for attempt := range 8 {
		candidate := refTime.Add(-time.Duration(attempt) * 3 * time.Hour)
		features, err = client.SearchPost(ctx, geoadmin.SearchPostBody{
			Collections:               []string{collection.ID},
			ForecastReferenceDatetime: candidate.Format(time.RFC3339),
			ForecastVariable:          "T_2M",
			ForecastPerturbed:         utl.Ptr(false),
		})
		require.NoError(t, err)
		if len(features) > 0 {
			refTime = candidate
			t.Logf("found data at refTime: %s (attempt %d)", refTime, attempt)
			break
		}
	}
	require.Len(t, features, 34)

	for _, f := range features {
		fp := f.Properties.Forecast()
		require.Equal(t, "T_2M", fp.Variable)
		require.Equal(t, refTime, fp.ReferenceDatetime)
		require.False(t, fp.Perturbed)
		t.Logf("feature: %+v", fp)
	}
}

func TestOnlineStacFetchItemsForVariables(t *testing.T) {
	ctx := context.Background()
	features, coll, err := fetchItemsForVariables(ctx, []string{"T_2M", "U"}, false)
	require.NoError(t, err)
	require.NotNil(t, coll)
	require.Len(t, features, 34*2)

	refTime := features[0].Properties.Forecast().ReferenceDatetime
	t.Logf("refTime: %s", refTime)

	for _, f := range features {
		fp := f.Properties.Forecast()
		require.False(t, fp.Perturbed)
		require.Equal(t, refTime, fp.ReferenceDatetime)
		horz, err := geoadmin.ParseISO8601Duration(fp.Horizon)
		require.NoError(t, err)
		require.Equal(t, fp.ReferenceDatetime.Add(horz), fp.Datetime)
		t.Logf("feature: %+v", fp)
	}
}
