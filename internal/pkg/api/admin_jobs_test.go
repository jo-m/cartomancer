package api_test

import (
	"database/sql"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"jo-m.ch/go/cartomancer/internal/pkg/db"
)

// seedJob inserts a row via [db.Queries.CreateJob] and then forces it into the
// requested terminal status with a raw UPDATE. This bypasses the state machine
// to keep tests independent of insertion order.
//
// Only these fields are read from j: CreatedAt, MaxAttempts, DelayS,
// BackoffFactorS, Kind, ArgsJson, Status, Attempts, StartedAt, FinishedAt,
// Error, Pid.
func seedJob(t *testing.T, d *db.DB, j db.Job) db.Job {
	t.Helper()
	created, err := d.QueryRW().CreateJob(t.Context(), db.CreateJobParams{
		CreatedAt:      j.CreatedAt,
		MaxAttempts:    j.MaxAttempts,
		DelayS:         j.DelayS,
		BackoffFactorS: j.BackoffFactorS,
		Kind:           j.Kind,
		ArgsJson:       j.ArgsJson,
	})
	require.NoError(t, err)
	if j.Status == "" || j.Status == "C" {
		return created
	}

	_, err = d.RW().ExecContext(t.Context(),
		`UPDATE jobs
		   SET status = ?, started_at = ?, finished_at = ?, attempts = ?, error = ?, pid = ?
		 WHERE id = ?`,
		j.Status, j.StartedAt, j.FinishedAt, j.Attempts, j.Error, j.Pid, created.ID,
	)
	require.NoError(t, err)

	rows, err := d.QueryRW().GetJobs(t.Context())
	require.NoError(t, err)
	for _, row := range rows {
		if row.ID == created.ID {
			return row
		}
	}
	t.Fatalf("seeded job %d not found", created.ID)
	return db.Job{}
}

func TestAdminListJobs_Empty(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("admin@example.com", "Admin", "adminpass", true)
	client := e.newClient()
	e.login(client, "admin@example.com", "adminpass")

	var resp map[string]any
	status, _ := e.do(client, http.MethodGet, "/admin/jobs", nil, &resp)
	assert.Equal(t, http.StatusOK, status)
	assert.Empty(t, resp["jobs"])
	assert.Empty(t, resp["byKind"])
	assert.Empty(t, resp["statusCounts"])
	assert.Equal(t, false, resp["truncated"])

	// The test env always starts a worker, so exactly one runner id exists.
	// Each entry must be a JSON string so JavaScript clients keep precision
	// for large random int64 ids.
	runnerIDs := resp["runnerIds"].([]any)
	require.Len(t, runnerIDs, 1)
	_, ok := runnerIDs[0].(string)
	assert.True(t, ok, "runnerIds entry must be a string, got %T", runnerIDs[0])
}

func TestAdminListJobs_Forbidden(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret11", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "secret11")

	status, _ := e.do(client, http.MethodGet, "/admin/jobs", nil, nil)
	assert.Equal(t, http.StatusForbidden, status)
}

func TestAdminListJobs_Unauthenticated(t *testing.T) {
	e := newTestEnv(t)
	client := e.newClient()
	status, _ := e.do(client, http.MethodGet, "/admin/jobs", nil, nil)
	assert.Equal(t, http.StatusUnauthorized, status)
}

func TestAdminListJobs_WithData(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("admin@example.com", "Admin", "adminpass", true)
	client := e.newClient()
	e.login(client, "admin@example.com", "adminpass")

	now := time.Now().UTC().Truncate(time.Second)

	// Seed a queued job with backoff and a non-zero delay so we can verify
	// nextAttemptAt is computed.
	queued := seedJob(t, e.d, db.Job{
		CreatedAt:      now,
		MaxAttempts:    3,
		DelayS:         60,
		BackoffFactorS: 0,
		Kind:           "test.kind.queued",
		ArgsJson:       `{"foo":"bar"}`,
	})
	// Seed a successful job (will appear in byKind / statusCounts).
	seedJob(t, e.d, db.Job{
		CreatedAt:   now,
		StartedAt:   sql.NullTime{Valid: true, Time: now},
		FinishedAt:  sql.NullTime{Valid: true, Time: now.Add(2 * time.Second)},
		Status:      "S",
		Attempts:    1,
		MaxAttempts: 1,
		Kind:        "test.kind.queued",
		ArgsJson:    `{"foo":"baz"}`,
	})
	// Seed a failed job that may retry. Pid is a large random int64 like the
	// real runtime values, used to verify the API serialises it as a string.
	const failedPid int64 = -5626924567125626000
	failed := seedJob(t, e.d, db.Job{
		CreatedAt:      now,
		StartedAt:      sql.NullTime{Valid: true, Time: now},
		FinishedAt:     sql.NullTime{Valid: true, Time: now.Add(3 * time.Second)},
		Status:         "E",
		Attempts:       1,
		MaxAttempts:    3,
		DelayS:         10,
		BackoffFactorS: 0,
		Kind:           "test.kind.failing",
		ArgsJson:       `{"x":1}`,
		Error:          sql.NullString{Valid: true, String: "boom"},
		Pid:            sql.NullInt64{Valid: true, Int64: failedPid},
	})

	var resp map[string]any
	statusCode, _ := e.do(client, http.MethodGet, "/admin/jobs", nil, &resp)
	require.Equal(t, http.StatusOK, statusCode)

	jobs := resp["jobs"].([]any)
	require.Len(t, jobs, 3)

	// Rows are ordered by id DESC: failed (last seeded) first.
	first := jobs[0].(map[string]any)
	assert.Equal(t, float64(failed.ID), first["id"])
	assert.Equal(t, "E", first["status"])
	assert.Equal(t, "boom", first["error"])
	assert.Contains(t, first, "nextAttemptAt")
	// runnerId is a string, not a number, so JS clients keep precision for
	// large random int64 worker ids.
	assert.Equal(t, "-5626924567125626000", first["runnerId"])

	// Verify the queued row carries a nextAttemptAt = createdAt + delayS.
	queuedRow := jobs[2].(map[string]any)
	assert.Equal(t, float64(queued.ID), queuedRow["id"])
	expectedNext := now.Add(60 * time.Second).Format(time.RFC3339)
	assert.Equal(t, expectedNext, queuedRow["nextAttemptAt"])
	assert.Equal(t, `{"foo":"bar"}`, queuedRow["argsJson"])

	// statusCounts: should sum to 3 across C/E/S.
	var total int64
	statusByCode := map[string]int64{}
	for _, raw := range resp["statusCounts"].([]any) {
		row := raw.(map[string]any)
		c := int64(row["count"].(float64))
		statusByCode[row["status"].(string)] = c
		total += c
	}
	assert.Equal(t, int64(3), total)
	assert.Equal(t, int64(1), statusByCode["C"])
	assert.Equal(t, int64(1), statusByCode["E"])
	assert.Equal(t, int64(1), statusByCode["S"])

	// byKind: two distinct kinds.
	kinds := map[string]map[string]any{}
	for _, raw := range resp["byKind"].([]any) {
		row := raw.(map[string]any)
		kinds[row["kind"].(string)] = row
	}
	require.Contains(t, kinds, "test.kind.queued")
	require.Contains(t, kinds, "test.kind.failing")
	assert.Equal(t, float64(2), kinds["test.kind.queued"]["total"])
	assert.Equal(t, float64(1), kinds["test.kind.queued"]["created"])
	assert.Equal(t, float64(1), kinds["test.kind.queued"]["succeeded"])
	assert.Equal(t, float64(1), kinds["test.kind.failing"]["total"])
	assert.Equal(t, float64(1), kinds["test.kind.failing"]["errored"])
}

func TestAdminListJobs_Filters(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("admin@example.com", "Admin", "adminpass", true)
	client := e.newClient()
	e.login(client, "admin@example.com", "adminpass")

	now := time.Now().UTC().Truncate(time.Second)
	seedJob(t, e.d, db.Job{
		CreatedAt: now, MaxAttempts: 1, Kind: "kind.a", ArgsJson: `{}`,
	})
	seedJob(t, e.d, db.Job{
		CreatedAt: now, MaxAttempts: 1, Kind: "kind.b", ArgsJson: `{}`,
	})
	seedJob(t, e.d, db.Job{
		CreatedAt:   now,
		StartedAt:   sql.NullTime{Valid: true, Time: now},
		FinishedAt:  sql.NullTime{Valid: true, Time: now.Add(time.Second)},
		Status:      "E",
		Attempts:    1,
		MaxAttempts: 1,
		Kind:        "kind.b",
		ArgsJson:    `{}`,
		Error:       sql.NullString{Valid: true, String: "fail"},
	})

	// Filter by kind.
	var resp map[string]any
	statusCode, _ := e.do(client, http.MethodGet, "/admin/jobs?kind=kind.b", nil, &resp)
	require.Equal(t, http.StatusOK, statusCode)
	jobs := resp["jobs"].([]any)
	assert.Len(t, jobs, 2)
	for _, j := range jobs {
		assert.Equal(t, "kind.b", j.(map[string]any)["kind"])
	}

	// Filter by status.
	statusCode, _ = e.do(client, http.MethodGet, "/admin/jobs?status=E", nil, &resp)
	require.Equal(t, http.StatusOK, statusCode)
	jobs = resp["jobs"].([]any)
	require.Len(t, jobs, 1)
	assert.Equal(t, "E", jobs[0].(map[string]any)["status"])

	// errorOnly=true returns only the failing row.
	statusCode, _ = e.do(client, http.MethodGet, "/admin/jobs?errorOnly=true", nil, &resp)
	require.Equal(t, http.StatusOK, statusCode)
	jobs = resp["jobs"].([]any)
	require.Len(t, jobs, 1)
	assert.Equal(t, "fail", jobs[0].(map[string]any)["error"])

	// Bad status -> 400.
	statusCode, _ = e.do(client, http.MethodGet, "/admin/jobs?status=BOGUS", nil, nil)
	assert.Equal(t, http.StatusBadRequest, statusCode)

	// Bad errorOnly -> 400.
	statusCode, _ = e.do(client, http.MethodGet, "/admin/jobs?errorOnly=maybe", nil, nil)
	assert.Equal(t, http.StatusBadRequest, statusCode)
}
