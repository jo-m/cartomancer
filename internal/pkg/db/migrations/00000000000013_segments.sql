-- +goose Up
CREATE TABLE segment_junctions (
    uuid TEXT PRIMARY KEY,
    h3_cell TEXT NOT NULL UNIQUE,
    lat REAL NOT NULL,
    lon REAL NOT NULL,
    created_at DATETIME NOT NULL
) WITHOUT ROWID;

CREATE TABLE segments (
    uuid TEXT PRIMARY KEY,
    start_junction_id TEXT NOT NULL REFERENCES segment_junctions(uuid) ON DELETE CASCADE,
    end_junction_id TEXT NOT NULL REFERENCES segment_junctions(uuid) ON DELETE CASCADE,
    h3_resolution INTEGER NOT NULL,
    distance_m REAL NOT NULL,
    ascent_m REAL NOT NULL,
    n_tracks INTEGER NOT NULL,
    polyline TEXT NOT NULL,
    created_at DATETIME NOT NULL
) WITHOUT ROWID;

CREATE TABLE segment_tracks (
    segment_id TEXT NOT NULL REFERENCES segments(uuid) ON DELETE CASCADE,
    track_id TEXT NOT NULL REFERENCES tracks(uuid) ON DELETE CASCADE,
    PRIMARY KEY (segment_id, track_id)
) WITHOUT ROWID;

-- +goose Down
DROP TABLE segment_tracks;
DROP TABLE segments;
DROP TABLE segment_junctions;
