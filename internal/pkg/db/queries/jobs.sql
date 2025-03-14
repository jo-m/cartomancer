-- name: CreateJob :one
INSERT INTO jobs (
  created_at, max_attempts, kind, args_json
) VALUES (
  ?, ?, ?, ?
)
RETURNING *;

-- name: SetNextJobRunning :one
UPDATE jobs
SET
  started_at = ?,
  attempts = (CASE WHEN (status = 'A') THEN (attempts) ELSE (attempts + 1) END),
  status = 'R',
  error = NULL
WHERE id = (
  SELECT id FROM jobs
  WHERE
    status IN ('C', 'A', 'E')
    AND attempts < max_attempts
  ORDER BY attempts ASC, created_at ASC, id ASC
  LIMIT 1
)
RETURNING *;

-- name: SetJobSuccess :exec
UPDATE jobs
SET finished_at = ?, status = 'S',  error = NULL
WHERE id = ? AND status = 'R';

-- name: SetJobError :exec
UPDATE jobs
SET finished_at = ?, status = 'E', error = ?
WHERE id = ? AND status = 'R';

-- name: SetJobsAborted :exec
UPDATE jobs
SET status = 'A', finished_at = ?, error = "Aborted"
WHERE status = 'R';

-- name: CleanupJobs :execrows
DELETE FROM jobs
WHERE
  status = 'S'
  OR (status IN ('E', 'A') AND attempts >= max_attempts);

-- name: SetJobRunnerProcessID :one
INSERT INTO job_runner_process_id (
  pid, random_id
) VALUES (
  ?, ?
)
RETURNING *;

-- name: GetJobs :many
SELECT * FROM jobs;

-- name: DeleteJobRunnerProcessIDs :exec
DELETE FROM job_runner_process_id
WHERE id != ?;
