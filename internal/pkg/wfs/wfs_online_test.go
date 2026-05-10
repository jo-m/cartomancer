//go:build online

package wfs_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"jo-m.ch/go/cartomancer/internal/pkg/wfs"
)

// testServerURL points at a live WFS endpoint used by the online tests.
const testServerURL = "https://maps.zh.ch/wfs/TbaBaustellenZHWFS"

func TestOnlineGetCapabilities(t *testing.T) {
	ctx := context.Background()
	c := wfs.NewClient(testServerURL)

	caps, err := c.GetCapabilities(ctx)
	require.NoError(t, err)
	require.Equal(t, "2.0.0", caps.Version)
	require.NotEmpty(t, caps.ServiceIdentification.Title)
	require.NotEmpty(t, caps.FeatureTypes)

	for _, ft := range caps.FeatureTypes {
		require.NotEmpty(t, ft.Name)
		require.NotEmpty(t, ft.DefaultCRS)
	}
}

func TestOnlineGetFeature(t *testing.T) {
	ctx := context.Background()
	c := wfs.NewClient(testServerURL)

	members, err := c.GetFeature(ctx, wfs.GetFeatureParams{
		TypeNames: "ms:baustellen-uebersicht",
		Count:     5,
	})
	require.NoError(t, err)
	// Pagination with Count=5 must produce more than one page worth of
	// features so we can confirm the 'next' link is being followed.
	require.Greater(t, len(members), 5, "should have paginated through multiple pages")

	for _, m := range members {
		require.NotEmpty(t, m.Feature.XMLName.Local)
		require.NotEmpty(t, m.Feature.GMLID)
		require.NotEmpty(t, m.Feature.InnerXML)
	}
}

func TestOnlineGetFeatureUnknownLayerReturnsException(t *testing.T) {
	ctx := context.Background()
	c := wfs.NewClient(testServerURL)

	_, err := c.GetFeature(ctx, wfs.GetFeatureParams{
		TypeNames: "ms:does-not-exist",
	})
	require.Error(t, err)

	var exc *wfs.ExceptionReport
	require.ErrorAs(t, err, &exc)
	require.NotEmpty(t, exc.Exceptions)
	require.Equal(t, "InvalidParameterValue", exc.Exceptions[0].Code)
}
