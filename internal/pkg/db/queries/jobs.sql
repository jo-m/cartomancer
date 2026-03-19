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

-- name: SetJobsAborted :execrows
UPDATE jobs
SET status = 'A', finished_at = ?, pid = NULL, error = "Aborted"
WHERE
  status = 'R'
  AND pid IS NOT NULL
  AND pid != @ourPID;

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

-- name: GetJobRunnerPIDs :many
SELECT * FROM job_runner_pid;
