package jobs

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"jo-m.ch/go/detour/internal/pkg/db"
	"jo-m.ch/go/detour/internal/pkg/logg"
)

func TestCleanerDeletesOnlyOldJobs(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()
	ctx := logg.WithTestLogger(t.Context(), t)

	q := d.QueryRW()

	// Job 1: finished 2 hours ago (should be cleaned with MinAge=1h).
	j1, err := q.CreateJob(ctx, db.CreateJobParams{
		CreatedAt:      time.Now().Add(-3 * time.Hour),
		MaxAttempts:    1,
		DelayS:         0,
		BackoffFactorS: 0,
		Kind:           "test.old",
		ArgsJson:       "{}",
	})
	require.NoError(t, err)
	_, err = q.SetNextJobRunning(ctx, db.SetNextJobRunningParams{
		StartedAt: sql.NullTime{Time: time.Now().Add(-3 * time.Hour), Valid: true},
		Pid:       sql.NullInt64{Int64: 1, Valid: true},
		Now:       time.Now(),
	})
	require.NoError(t, err)
	_, err = q.SetJobSuccess(ctx, db.SetJobSuccessParams{
		FinishedAt: sql.NullTime{Time: time.Now().Add(-2 * time.Hour), Valid: true},
		ID:         j1.ID,
	})
	require.NoError(t, err)

	// Job 2: finished 5 minutes ago (should be kept with MinAge=1h).
	j2, err := q.CreateJob(ctx, db.CreateJobParams{
		CreatedAt:      time.Now().Add(-10 * time.Minute),
		MaxAttempts:    1,
		DelayS:         0,
		BackoffFactorS: 0,
		Kind:           "test.recent",
		ArgsJson:       "{}",
	})
	require.NoError(t, err)
	_, err = q.SetNextJobRunning(ctx, db.SetNextJobRunningParams{
		StartedAt: sql.NullTime{Time: time.Now().Add(-10 * time.Minute), Valid: true},
		Pid:       sql.NullInt64{Int64: 1, Valid: true},
		Now:       time.Now(),
	})
	require.NoError(t, err)
	_, err = q.SetJobSuccess(ctx, db.SetJobSuccessParams{
		FinishedAt: sql.NullTime{Time: time.Now().Add(-5 * time.Minute), Valid: true},
		ID:         j2.ID,
	})
	require.NoError(t, err)

	// Run cleaner with MinAge=1h: should only delete job finished >= 1h ago.
	c := &cleaner{d: d}
	err = c.Run(ctx, cleanerArgs{MinAge: time.Hour})
	require.NoError(t, err)

	// Only the recent job should remain.
	jobs, err := q.GetJobs(ctx)
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	assert.Equal(t, j2.ID, jobs[0].ID)
}
