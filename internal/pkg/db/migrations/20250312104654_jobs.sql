-- +goose Up
-- +goose StatementBegin
CREATE TABLE jobs (
    id INTEGER PRIMARY KEY,

    created_at DATETIME NOT NULL,
    started_at DATETIME,
    finished_at DATETIME,

    -- C: Created, R: Running, A: Aborted, E: Error, S: Success
    status TEXT CHECK(status IN ('C', 'R', 'A', 'E', 'S') ) NOT NULL DEFAULT 'C',
    pid INTEGER DEFAULT NULL,
    delay_seconds INTEGER NOT NULL,
    -- Is increased as soon as the job switches to the 'R' status.
    attempts INTEGER NOT NULL DEFAULT 0,
    max_attempts INTEGER NOT NULL,

    kind TEXT NOT NULL,
    args_json TEXT NOT NULL,
    error TEXT
);
CREATE TABLE job_runner_pid (
    pid INTEGER NOT NULL,
    UNIQUE (pid)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE jobs;
DROP TABLE job_runner_pid;
-- +goose StatementEnd
