package jobs

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestJobsConfigValidate(t *testing.T) {
	t.Run("valid with cleanup", func(t *testing.T) {
		c := JobsConfig{
			AutoCleanupPeriod: time.Minute,
			AutoCleanupMinAge: 5 * time.Minute,
		}
		require.NoError(t, c.Validate())
	})

	t.Run("valid with cleanup disabled", func(t *testing.T) {
		c := JobsConfig{
			AutoCleanupPeriod: 0,
			AutoCleanupMinAge: 0,
		}
		require.NoError(t, c.Validate())
	})

	t.Run("negative cleanup period", func(t *testing.T) {
		c := JobsConfig{AutoCleanupPeriod: -1}
		require.ErrorContains(t, c.Validate(), "JOBS_AUTO_CLEANUP_PERIOD")
	})

	t.Run("negative cleanup min age", func(t *testing.T) {
		c := JobsConfig{AutoCleanupMinAge: -1}
		require.ErrorContains(t, c.Validate(), "JOBS_AUTO_CLEANUP_MIN_AGE")
	})

	t.Run("cleanup enabled but zero min age", func(t *testing.T) {
		c := JobsConfig{
			AutoCleanupPeriod: time.Minute,
			AutoCleanupMinAge: 0,
		}
		require.ErrorContains(t, c.Validate(), "JOBS_AUTO_CLEANUP_MIN_AGE")
	})
}
