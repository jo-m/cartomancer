//go:build online

package cols

import (
	"testing"

	"github.com/franiglesias/golden"
	"github.com/stretchr/testify/require"
	"jo-m.ch/go/cartomancer/internal/pkg/utl"
)

const (
	readmeURL       = BaseURL + "/readme.txt"
	featureCodesURL = BaseURL + "/featureCodes_en.txt"
)

// TestOnlineDownloadReadme downloads the GeoNames readme.txt and verifies it
// against a golden snapshot so that column format changes are detected.
func TestOnlineDownloadReadme(t *testing.T) {
	data, err := utl.DownloadFile(t.Context(), readmeURL)
	require.NoError(t, err)
	require.NotEmpty(t, data)

	golden.Verify(t, string(data), golden.Extension(".txt")) // golden.WaitApproval()
}

// TestOnlineDownloadFeatureCodes downloads the GeoNames featureCodes_en.txt
// and verifies it against a golden snapshot so that format changes are detected.
func TestOnlineDownloadFeatureCodes(t *testing.T) {
	data, err := utl.DownloadFile(t.Context(), featureCodesURL)
	require.NoError(t, err)
	require.NotEmpty(t, data)

	golden.Verify(t, string(data), golden.Extension(".txt")) // golden.WaitApproval()
}
