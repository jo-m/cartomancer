package jobs

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"jo-m.ch/go/cartomancer/internal/pkg/blob"
	"jo-m.ch/go/cartomancer/internal/pkg/db"
	"jo-m.ch/go/cartomancer/internal/pkg/load"
	"jo-m.ch/go/cartomancer/internal/pkg/logg"
	"jo-m.ch/go/cartomancer/internal/pkg/track"
)

// BackfillPreviewPolylineArgs are the arguments for the preview polyline
// backfill job. The job processes a single track per run and reschedules
// itself if more remain, so the args struct intentionally has no fields.
type BackfillPreviewPolylineArgs struct{}

// Kind implements [Args].
func (BackfillPreviewPolylineArgs) Kind() string { return "track.backfillPreviewPolyline" }

var _ Args = (*BackfillPreviewPolylineArgs)(nil)

// backfillRescheduleDelay is the delay applied when the job reschedules
// itself after successfully processing one track. Whole seconds are
// required by the job queue.
const backfillRescheduleDelay = 1 * time.Second

// BackfillPreviewPolylineHandler computes and stores preview polylines for
// tracks that still have a NULL preview_polyline column. Use
// [NewBackfillPreviewPolyline] to construct.
type BackfillPreviewPolylineHandler struct {
	d *db.DB
	s *Submitter
}

// NewBackfillPreviewPolyline creates a new backfill handler.
//
// Parameters:
//   - d: main database with the tracks and blobs tables.
//   - s: job submitter, used by the handler to reschedule itself while
//     more tracks still need processing.
func NewBackfillPreviewPolyline(d *db.DB, s *Submitter) *BackfillPreviewPolylineHandler {
	return &BackfillPreviewPolylineHandler{d: d, s: s}
}

var _ Job[BackfillPreviewPolylineArgs] = (*BackfillPreviewPolylineHandler)(nil)

// Run implements [Job]. It picks the oldest track without a preview
// polyline, parses its blob, computes the simplified encoded polyline and
// stores it. If at least one more track still needs backfilling, the job
// reschedules itself.
func (h *BackfillPreviewPolylineHandler) Run(ctx context.Context, _ BackfillPreviewPolylineArgs) error {
	uuid, err := h.d.QueryRO().NextTrackMissingPreviewPolyline(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		logg.Debug(ctx, "preview polyline backfill: nothing to do")
		return nil
	}
	if err != nil {
		return fmt.Errorf("next missing preview polyline: %w", err)
	}

	if err := h.processTrack(ctx, uuid); err != nil {
		return err
	}

	// Reschedule unless this was the last one.
	remaining, err := h.d.QueryRO().CountTracksMissingPreviewPolyline(ctx)
	if err != nil {
		return fmt.Errorf("count missing preview polylines: %w", err)
	}
	if remaining == 0 {
		logg.Info(ctx, "preview polyline backfill: completed")
		return nil
	}

	if err := Submit(ctx, h.s, BackfillPreviewPolylineArgs{}, Params{
		DelayS: backfillRescheduleDelay,
	}); err != nil {
		return fmt.Errorf("reschedule backfill: %w", err)
	}
	return nil
}

// processTrack computes the preview polyline for a single track and
// persists it. Tracks whose blobs cannot be parsed are skipped with an
// empty polyline so the backfill makes monotonic progress.
func (h *BackfillPreviewPolylineHandler) processTrack(ctx context.Context, uuid string) error {
	t, err := h.d.QueryRO().GetTrackByUUID(ctx, uuid)
	if err != nil {
		return fmt.Errorf("get track %s: %w", uuid, err)
	}

	encoded, err := computePreviewPolyline(ctx, h.d.QueryRO(), t)
	if err != nil {
		logg.Error(ctx, "preview polyline backfill: skipping track", "trackId", uuid, "err", err)
		// Persist an empty polyline so the row drops out of the candidate
		// set and we make forward progress on subsequent runs.
		encoded = ""
	}

	if err := h.d.QueryRW().SetTrackPreviewPolyline(ctx, db.SetTrackPreviewPolylineParams{
		Uuid: uuid,
		PreviewPolyline: sql.NullString{
			Valid:  true,
			String: encoded,
		},
	}); err != nil {
		return fmt.Errorf("set preview polyline %s: %w", uuid, err)
	}

	logg.Debug(ctx, "preview polyline backfilled", "trackId", uuid, "bytes", len(encoded))
	return nil
}

// computePreviewPolyline loads the track blob, parses it, simplifies the
// point sequence and returns the encoded polyline.
func computePreviewPolyline(ctx context.Context, q *db.Queries, t db.Track) (string, error) {
	b, err := blob.Get(ctx, q, t.BlobID)
	if err != nil {
		return "", fmt.Errorf("get blob: %w", err)
	}
	src, err := load.Blob(t.OriginalFilename, bytes.NewReader(b.Content))
	if err != nil {
		return "", fmt.Errorf("parse blob: %w", err)
	}
	tr, err := track.New(src)
	if err != nil {
		return "", fmt.Errorf("new track: %w", err)
	}
	simplified := tr.Points().SimplifyDP(track.PreviewPolylineEpsilonM)
	return track.EncodePolyline(simplified), nil
}

// EnqueueBackfillPreviewPolylineIfNeeded enqueues a single backfill job
// when at least one track still lacks a preview polyline. Safe to call on
// every startup; the job itself reschedules until everything is processed.
func EnqueueBackfillPreviewPolylineIfNeeded(ctx context.Context, d *db.DB, s *Submitter) error {
	n, err := d.QueryRO().CountTracksMissingPreviewPolyline(ctx)
	if err != nil {
		return fmt.Errorf("count missing preview polylines: %w", err)
	}
	if n == 0 {
		return nil
	}
	logg.Info(ctx, "preview polyline backfill: enqueueing", "remaining", n)
	return Submit(ctx, s, BackfillPreviewPolylineArgs{}, Params{})
}
