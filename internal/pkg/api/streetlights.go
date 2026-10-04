package api

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"jo-m.ch/go/cartomancer/internal/pkg/logg"
	"jo-m.ch/go/cartomancer/internal/pkg/streetlights"
)

// litStretchResponse is one lit section of a track, given as cumulative
// distances in meters from the start of the track.
type litStretchResponse struct {
	StartDistanceM float64 `json:"startDistanceM"`
	EndDistanceM   float64 `json:"endDistanceM"`
}

// trackStreetlightsResponse is the response body of the track streetlights
// endpoint.
type trackStreetlightsResponse struct {
	TotalDistanceM float64               `json:"totalDistanceM"`
	LitDistanceM   float64               `json:"litDistanceM"`
	LitFraction    float64               `json:"litFraction"`
	SampleStepM    float64               `json:"sampleStepM"`
	LitRadiusM     float64               `json:"litRadiusM"`
	Stretches      []litStretchResponse  `json:"stretches"`
	Attributions   []attributionResponse `json:"attributions"`
}

// handleGetTrackStreetlights returns how much of a track path is lit by
// streetlights, and where. The lit sections are reported as distance ranges
// into the track's dp5m polyline, which the client already downloads from the
// points endpoint. The etag combines the track and the streetlight corpus
// fingerprints, so it invalidates both on track edits and on lamp refreshes.
func (sv *server) handleGetTrackStreetlights(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	trackUUID := chi.URLParam(r, "uuid")

	t, ok := sv.getViewableTrack(w, r, trackUUID)
	if !ok {
		return
	}

	lampCount, lampsUpdatedAtMs, err := sv.d.GetStreetlightDataFingerprint(ctx)
	if err != nil {
		logg.Error(ctx, "failed to get streetlight data fingerprint", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	eTag := fmt.Sprintf(`"%d-%d-%d-v1"`, t.UpdatedAt.UnixMilli(), lampsUpdatedAtMs, lampCount)
	if r.Header.Get(headerIfNoneMatch) == eTag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	cov, err := streetlights.ComputeTrackCoverage(ctx, sv.d, t)
	if err != nil {
		logg.Error(ctx, "failed to compute track streetlight coverage", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	litFraction := 0.0
	if cov.TotalDistanceM > 0 {
		litFraction = cov.LitDistanceM / cov.TotalDistanceM
	}

	stretches := make([]litStretchResponse, 0, len(cov.Stretches))
	for _, s := range cov.Stretches {
		stretches = append(stretches, litStretchResponse{StartDistanceM: s.StartDistanceM, EndDistanceM: s.EndDistanceM})
	}
	attributions := make([]attributionResponse, 0, len(cov.Attributions))
	for _, a := range cov.Attributions {
		attributions = append(attributions, attributionResponse{Text: a.Text, Href: a.Href})
	}

	w.Header().Set(headerCacheControl, "private, max-age=3600")
	w.Header().Set(headerETag, eTag)
	writeJSON(w, http.StatusOK, trackStreetlightsResponse{
		TotalDistanceM: cov.TotalDistanceM,
		LitDistanceM:   cov.LitDistanceM,
		LitFraction:    litFraction,
		SampleStepM:    streetlights.SampleStepM,
		LitRadiusM:     streetlights.LitRadiusM,
		Stretches:      stretches,
		Attributions:   attributions,
	})
}
