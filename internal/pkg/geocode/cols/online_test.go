//go:build online

package cols

import (
	"testing"

	"github.com/franiglesias/golden"
	"github.com/stretchr/testify/require"
	"jo-m.ch/go/detour/internal/pkg/utl"
)

const readmeURL = BaseURL + "/readme.txt"

// TestOnlineDownloadReadme downloads the GeoNames readme.txt and verifies it
// against a golden snapshot so that column format changes are detected.
func TestOnlineDownloadReadme(t *testing.T) {
	data, err := utl.DownloadFile(readmeURL)
	require.NoError(t, err)
	require.NotEmpty(t, data)

	golden.Verify(t, string(data), golden.Extension(".txt")) // golden.WaitApproval()
}
