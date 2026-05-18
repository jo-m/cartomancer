package forecastdb

import (
	"context"
	"fmt"
	"time"
)

// CreateForecastFileParams are the fields needed to create a forecast file
// (metadata row plus its GRIB2 blob payload).
type CreateForecastFileParams struct {
	ValidTime      time.Time
	ValidUntilTime time.Time
	Variable       string
	File           []byte
	ForecastID     int64
}

// CreateForecastFile inserts the metadata row for a forecast file and its
// associated GRIB2 blob payload. Both writes must occur in the same SQLite
// transaction so that the FK from forecast_file_blobs to forecast_files is
// satisfied; callers should therefore invoke this through [DB.WithTx] (or
// pass a *Queries bound to a transaction). The returned [ForecastFile]
// holds the inserted metadata row (without the blob).
func (q *Queries) CreateForecastFile(ctx context.Context, arg CreateForecastFileParams) (ForecastFile, error) {
	meta, err := q.CreateForecastFileMeta(ctx, CreateForecastFileMetaParams{
		ValidTime:      arg.ValidTime,
		ValidUntilTime: arg.ValidUntilTime,
		Variable:       arg.Variable,
		FileSize:       int64(len(arg.File)),
		ForecastID:     arg.ForecastID,
	})
	if err != nil {
		return ForecastFile{}, fmt.Errorf("insert forecast_files metadata: %w", err)
	}

	if err := q.CreateForecastFileBlob(ctx, CreateForecastFileBlobParams{
		ForecastFileID: meta.ID,
		File:           arg.File,
	}); err != nil {
		return ForecastFile{}, fmt.Errorf("insert forecast_file_blobs payload: %w", err)
	}

	return meta, nil
}
