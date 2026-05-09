package api

import (
	"math"
	"net/http"
	"strconv"
	"time"

	"jo-m.ch/go/cartomancer/internal/pkg/db"
	"jo-m.ch/go/cartomancer/internal/pkg/logg"
)

// adminJobsListMaxLimit is the upper bound for the limit query parameter.
const adminJobsListMaxLimit = 1000

// adminJobsListDefaultLimit is the default page size when no limit is given.
const adminJobsListDefaultLimit = 200

// adminJobResponse is a single job row returned by the admin endpoint.
// Time fields are RFC3339-formatted strings (or nil when not set).
//
// RunnerID identifies the worker process that picked the job up; despite the
// `pid` column name in the database, it is a random int64 (see
// jobs.singleinstance.go), so we serialize it as a string to avoid losing
// precision in JavaScript clients.
type adminJobResponse struct {
	ID             int64   `json:"id"`
	Kind           string  `json:"kind"`
	Status         string  `json:"status"`
	CreatedAt      string  `json:"createdAt"`
	StartedAt      *string `json:"startedAt,omitempty"`
	FinishedAt     *string `json:"finishedAt,omitempty"`
	NextAttemptAt  *string `json:"nextAttemptAt,omitempty"`
	RunnerID       *string `json:"runnerId,omitempty"`
	Attempts       int64   `json:"attempts"`
	MaxAttempts    int64   `json:"maxAttempts"`
	DelayS         int64   `json:"delayS"`
	BackoffFactorS int64   `json:"backoffFactorS"`
	ArgsJSON       string  `json:"argsJson"`
	Error          *string `json:"error,omitempty"`
}

// adminJobStatusCount carries one status -> count entry of the global status histogram.
type adminJobStatusCount struct {
	Status string `json:"status"`
	Count  int64  `json:"count"`
}

// adminJobKindRow summarises a single job kind across the whole table.
type adminJobKindRow struct {
	Kind      string `json:"kind"`
	Total     int64  `json:"total"`
	Created   int64  `json:"created"`
	Running   int64  `json:"running"`
	Succeeded int64  `json:"succeeded"`
	Errored   int64  `json:"errored"`
	Aborted   int64  `json:"aborted"`
}

// adminJobsResponse is the full payload returned by GET /admin/jobs.
//
// RunnerIDs is the list of currently registered worker process IDs (random
// int64 values, see jobs.singleinstance.go). Serialized as strings to avoid
// JavaScript number-precision loss; expect exactly one entry under normal
// operation.
type adminJobsResponse struct {
	RunnerIDs    []string              `json:"runnerIds"`
	StatusCounts []adminJobStatusCount `json:"statusCounts"`
	ByKind       []adminJobKindRow     `json:"byKind"`
	Jobs         []adminJobResponse    `json:"jobs"`
	// Limit is the maximum number of rows returned in [Jobs].
	Limit int64 `json:"limit"`
	// Truncated is true when [Jobs] hit [Limit] and more rows may exist.
	Truncated bool `json:"truncated"`
}

// nextAttemptAt returns the timestamp at which the next attempt of the given
// job is scheduled, or nil if the job is finished or has no remaining attempts.
//
// The formula mirrors the one in the SetNextJobRunning / SetJobError SQL
// queries: created_at + delay_s + backoff_factor_s * (2^attempts - 1) seconds.
func nextAttemptAt(j db.Job) *time.Time {
	switch j.Status {
	case "C", "A", "E":
	default:
		return nil
	}
	if j.Attempts >= j.MaxAttempts {
		return nil
	}
	backoff := j.DelayS + j.BackoffFactorS*(int64(math.Pow(2, float64(j.Attempts)))-1)
	t := j.CreatedAt.Add(time.Duration(backoff) * time.Second)
	return &t
}

// adminJobResponseFromDB converts a [db.Job] row to its API representation.
func adminJobResponseFromDB(j db.Job) adminJobResponse {
	resp := adminJobResponse{
		ID:             j.ID,
		Kind:           j.Kind,
		Status:         j.Status,
		CreatedAt:      j.CreatedAt.Format(time.RFC3339),
		Attempts:       j.Attempts,
		MaxAttempts:    j.MaxAttempts,
		DelayS:         j.DelayS,
		BackoffFactorS: j.BackoffFactorS,
		ArgsJSON:       j.ArgsJson,
	}
	if j.StartedAt.Valid {
		s := j.StartedAt.Time.Format(time.RFC3339)
		resp.StartedAt = &s
	}
	if j.FinishedAt.Valid {
		s := j.FinishedAt.Time.Format(time.RFC3339)
		resp.FinishedAt = &s
	}
	if j.Pid.Valid {
		s := strconv.FormatInt(j.Pid.Int64, 10)
		resp.RunnerID = &s
	}
	if j.Error.Valid {
		s := j.Error.String
		resp.Error = &s
	}
	if next := nextAttemptAt(j); next != nil {
		s := next.Format(time.RFC3339)
		resp.NextAttemptAt = &s
	}
	return resp
}

// validJobStatusFilter reports whether s is an accepted value for the status
// query parameter (one of the on-disk status codes, or empty for "no filter").
func validJobStatusFilter(s string) bool {
	switch s {
	case "", "C", "R", "A", "E", "S":
		return true
	}
	return false
}

// handleAdminListJobs returns a snapshot of the persistent job queue: live
// runner PIDs, global status histogram, per-kind summary, and the most recent
// job rows matching the optional kind / status / errorOnly filters.
//
// Auto-cleanup means succeeded and exhausted jobs disappear from the table
// shortly after they finish (see jobs.JobsConfig.AutoCleanupMinAge). The
// dashboard only ever shows what is currently in the database.
func (sv *server) handleAdminListJobs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ro := sv.d.QueryRO()

	q := r.URL.Query()
	kind := q.Get("kind")
	status := q.Get("status")
	if !validJobStatusFilter(status) {
		writeError(w, http.StatusBadRequest, "invalid status filter")
		return
	}

	var errorOnly int64
	if v := q.Get("errorOnly"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid errorOnly value")
			return
		}
		if b {
			errorOnly = 1
		}
	}

	limit := int64(adminJobsListDefaultLimit)
	if v := q.Get("limit"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil || n <= 0 {
			writeError(w, http.StatusBadRequest, "invalid limit value")
			return
		}
		if n > adminJobsListMaxLimit {
			n = adminJobsListMaxLimit
		}
		limit = n
	}

	rawPids, err := ro.GetJobRunnerPIDs(ctx)
	if err != nil {
		logg.Error(ctx, "failed to list job runner ids", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}
	runnerIDs := make([]string, len(rawPids))
	for i, p := range rawPids {
		runnerIDs[i] = strconv.FormatInt(p, 10)
	}

	statusRows, err := ro.AdminCountJobsByStatus(ctx)
	if err != nil {
		logg.Error(ctx, "failed to count jobs by status", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}
	statusCounts := make([]adminJobStatusCount, len(statusRows))
	for i, row := range statusRows {
		statusCounts[i] = adminJobStatusCount{Status: row.Status, Count: row.Count}
	}

	kindRows, err := ro.AdminListJobKinds(ctx)
	if err != nil {
		logg.Error(ctx, "failed to list job kinds", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}
	byKind := make([]adminJobKindRow, len(kindRows))
	for i, row := range kindRows {
		byKind[i] = adminJobKindRow{
			Kind:      row.Kind,
			Total:     row.Total,
			Created:   row.Created,
			Running:   row.Running,
			Succeeded: row.Succeeded,
			Errored:   row.Errored,
			Aborted:   row.Aborted,
		}
	}

	jobs, err := ro.AdminListJobs(ctx, db.AdminListJobsParams{
		Kind:      kind,
		Status:    status,
		ErrorOnly: errorOnly,
		Limit:     limit,
	})
	if err != nil {
		logg.Error(ctx, "failed to list jobs", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}
	resp := make([]adminJobResponse, len(jobs))
	for i, j := range jobs {
		resp[i] = adminJobResponseFromDB(j)
	}

	writeJSON(w, http.StatusOK, adminJobsResponse{
		RunnerIDs:    runnerIDs,
		StatusCounts: statusCounts,
		ByKind:       byKind,
		Jobs:         resp,
		Limit:        limit,
		Truncated:    int64(len(jobs)) >= limit,
	})
}
