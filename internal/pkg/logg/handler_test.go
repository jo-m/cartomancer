package logg

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoggConfigValidate(t *testing.T) {
	c := LoggConfig{}
	require.NoError(t, c.Validate())
}
