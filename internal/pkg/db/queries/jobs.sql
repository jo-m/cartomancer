-- name: CreateJob :one
INSERT INTO jobs (
  created_at, max_attempts, delay_s, backoff_factor_s, kind, args_json
) VALUES (
  ?, ?, ?, ?, ?, ?
)
RETURNING *;

-- name: SetNextJobRunning :one
UPDATE jobs
SET
  started_at = ?,
  pid = ?,
  attempts = (CASE WHEN (status = 'A') THEN (attempts) ELSE (attempts + 1) END),
  status = 'R',
  error = NULL
WHERE id = (
  SELECT id FROM jobs
  WHERE
    status IN ('C', 'A', 'E')
    AND attempts < max_attempts
    AND Datetime(@now) >= Datetime(jobs.created_at, (delay_s + backoff_factor_s * (Power(2,jobs.attempts)-1)) || ' seconds')
  ORDER BY attempts ASC, created_at ASC, id ASC
  LIMIT 1
)
RETURNING *;

-- name: SetJobSuccess :execrows
UPDATE jobs
SET status = 'S', finished_at = ?, pid = NULL, error = NULL
WHERE id = ? AND status = 'R';

-- name: SetJobError :one
UPDATE jobs
SET status = 'E', finished_at = ?, pid = NULL, error = ?
WHERE id = ? AND status = 'R'
RETURNING
  CASE WHEN attempts < max_attempts THEN
    Datetime(jobs.created_at, (delay_s + backoff_factor_s * (Power(2,jobs.attempts)-1)) || ' seconds')
  ELSE
    NULL
  END
  AS next_attempt_at
;

-- name: SetJobsAborted :many
UPDATE jobs
SET status = 'A', finished_at = ?, pid = NULL, error = "Aborted"
WHERE
  status = 'R'
  AND pid IS NOT NULL
  AND pid != @ourPID
RETURNING id, kind;

-- name: CleanupJobs :execrows
DELETE FROM jobs
WHERE
  (status = 'S' OR (status IN ('E', 'A') AND attempts >= max_attempts))
  AND Datetime(jobs.finished_at) <= Datetime(@maxFinishedAt);

-- name: GetJobs :many
SELECT * FROM jobs;

-- name: InsertJobRunnerPID :exec
INSERT INTO job_runner_pid (
  pid
) VALUES (
  ?
);

-- name: DeleteOtherJobRunnerPIDs :exec
DELETE FROM job_runner_pid
WHERE pid != @ourPID;

-- name: HasActiveJob :one
SELECT EXISTS(
  SELECT 1 FROM jobs
  WHERE kind = ? AND args_json = ? AND status IN ('C', 'R', 'A', 'E')
  AND attempts < max_attempts
) AS active;

-- name: HasRecentActiveJob :one
SELECT EXISTS(
  SELECT 1 FROM jobs
  WHERE kind = ? AND args_json = ? AND status IN ('C', 'R', 'A', 'E')
  AND attempts < max_attempts
  AND Datetime(jobs.created_at) >= Datetime(@since)
) AS active;

-- name: GetJobRunnerPIDs :many
SELECT * FROM job_runner_pid;

-- name: AdminListJobs :many
SELECT * FROM jobs
WHERE
  (@kind = '' OR kind = @kind)
  AND (@status = '' OR status = @status)
  AND (@errorOnly = 0 OR error IS NOT NULL)
ORDER BY id DESC
LIMIT @limit;

-- name: AdminCountJobsByStatus :many
SELECT status, COUNT(*) AS count
FROM jobs
GROUP BY status;

-- name: AdminListJobKinds :many
SELECT
  kind,
  COUNT(*) AS total,
  CAST(COALESCE(SUM(CASE WHEN status = 'C' THEN 1 ELSE 0 END), 0) AS INTEGER) AS created,
  CAST(COALESCE(SUM(CASE WHEN status = 'R' THEN 1 ELSE 0 END), 0) AS INTEGER) AS running,
  CAST(COALESCE(SUM(CASE WHEN status = 'S' THEN 1 ELSE 0 END), 0) AS INTEGER) AS succeeded,
  CAST(COALESCE(SUM(CASE WHEN status = 'E' THEN 1 ELSE 0 END), 0) AS INTEGER) AS errored,
  CAST(COALESCE(SUM(CASE WHEN status = 'A' THEN 1 ELSE 0 END), 0) AS INTEGER) AS aborted
FROM jobs
GROUP BY kind
ORDER BY kind;
