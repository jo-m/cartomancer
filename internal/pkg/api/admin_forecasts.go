package api

import (
	"net/http"
	"time"

	"jo-m.ch/go/cartomancer/internal/pkg/logg"
)

type adminForecastFileResponse struct {
	ID             int64  `json:"id"`
	ValidTime      string `json:"validTime"`
	ValidUntilTime string `json:"validUntilTime"`
	Variable       string `json:"variable"`
	FileSize       int64  `json:"fileSize"`
}

type adminForecastResponse struct {
	ID            int64                       `json:"id"`
	CreatedAt     string                      `json:"createdAt"`
	ReferenceTime string                      `json:"referenceTime"`
	Bounds        *bboxResponse               `json:"bounds,omitempty"`
	Attribution   attributionResponse         `json:"attribution"`
	Files         []adminForecastFileResponse `json:"files"`
}

type adminForecastsResponse struct {
	Forecasts []adminForecastResponse `json:"forecasts"`
}

// handleAdminListForecasts returns all forecasts with their files (excluding blob data).
func (sv *server) handleAdminListForecasts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	rows, err := sv.d.QueryRO().ListForecastsWithFiles(ctx)
	if err != nil {
		logg.Error(ctx, "failed to list forecasts", "err", err)
		writeError(w, http.StatusInternalServerError, "failed to list forecasts")
		return
	}

	// Group the flat joined rows into nested forecast responses.
	var forecasts []adminForecastResponse
	var current *adminForecastResponse
	for _, row := range rows {
		if current == nil || current.ID != row.ForecastID {
			if current != nil {
				forecasts = append(forecasts, *current)
			}
			fc := adminForecastResponse{
				ID:            row.ForecastID,
				CreatedAt:     row.CreatedAt.Format(time.RFC3339),
				ReferenceTime: row.ReferenceTime.Format(time.RFC3339),
				Bounds:        nullBBox(row.BoundsMinLat, row.BoundsMinLon, row.BoundsMaxLat, row.BoundsMaxLon),
				Attribution:   attributionResponse{Text: row.Attribution, Href: row.AttributionHref},
				Files:         []adminForecastFileResponse{},
			}
			current = &fc
		}
		if row.FileID.Valid {
			current.Files = append(current.Files, adminForecastFileResponse{
				ID:             row.FileID.Int64,
				ValidTime:      row.ValidTime.Time.Format(time.RFC3339),
				ValidUntilTime: row.ValidUntilTime.Time.Format(time.RFC3339),
				Variable:       row.Variable.String,
				FileSize:       row.FileSize.Int64,
			})
		}
	}
	if current != nil {
		forecasts = append(forecasts, *current)
	}
	if forecasts == nil {
		forecasts = []adminForecastResponse{}
	}

	writeJSON(w, http.StatusOK, adminForecastsResponse{Forecasts: forecasts})
}
