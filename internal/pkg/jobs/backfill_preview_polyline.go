package jobs

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"

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
const backfillRescheduleDelay = 0

// BackfillPreviewPolylineHandler computes and stores preview polylines for
// tracks that still have NULL polyline_dp5m_varint or polyline_dp50m_varint
// columns. Use [NewBackfillPreviewPolyline] to construct.
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

// Run implements [Job]. It picks the oldest track without preview
// polylines, parses its blob, computes both simplified varint-encoded
// polylines (5 m and 50 m epsilon) and stores them. If at least one more
// track still needs backfilling, the job reschedules itself.
func (h *BackfillPreviewPolylineHandler) Run(ctx context.Context, _ BackfillPreviewPolylineArgs) error {
	uuid, err := h.d.QueryRO().NextTrackMissingPreviewPolylines(ctx)
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
	remaining, err := h.d.QueryRO().CountTracksMissingPreviewPolylines(ctx)
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

// processTrack computes both preview polylines for a single track and
// persists them. Tracks whose blobs cannot be parsed are skipped with empty
// polylines so the backfill makes monotonic progress.
func (h *BackfillPreviewPolylineHandler) processTrack(ctx context.Context, uuid string) error {
	t, err := h.d.QueryRO().GetTrackByUUID(ctx, uuid)
	if err != nil {
		return fmt.Errorf("get track %s: %w", uuid, err)
	}

	dp5, dp50, err := computePreviewPolylines(ctx, h.d.QueryRO(), t)
	if err != nil {
		logg.Error(ctx, "preview polyline backfill: skipping track", "trackId", uuid, "err", err)
		// Persist empty polylines so the row drops out of the candidate
		// set and we make forward progress on subsequent runs.
		dp5, dp50 = []byte{}, []byte{}
	}

	if err := h.d.QueryRW().SetTrackPreviewPolylines(ctx, db.SetTrackPreviewPolylinesParams{
		Uuid:                uuid,
		PolylineDp5mVarint:  dp5,
		PolylineDp50mVarint: dp50,
	}); err != nil {
		return fmt.Errorf("set preview polylines %s: %w", uuid, err)
	}

	logg.Debug(ctx, "preview polylines backfilled", "trackId", uuid, "dp5mBytes", len(dp5), "dp50mBytes", len(dp50))
	return nil
}

// computePreviewPolylines loads the track blob, parses it, and returns the
// 5 m and 50 m simplified polylines encoded as varint byte slices.
func computePreviewPolylines(ctx context.Context, q *db.Queries, t db.Track) ([]byte, []byte, error) {
	b, err := blob.Get(ctx, q, t.BlobID)
	if err != nil {
		return nil, nil, fmt.Errorf("get blob: %w", err)
	}
	src, err := load.Blob(t.OriginalFilename, bytes.NewReader(b.Content))
	if err != nil {
		return nil, nil, fmt.Errorf("parse blob: %w", err)
	}
	tr, err := track.New(src)
	if err != nil {
		return nil, nil, fmt.Errorf("new track: %w", err)
	}
	pts := tr.Points()
	dp5, err := track.EncodeVarint(pts.SimplifyDP(track.PreviewPolylineEpsilon5M))
	if err != nil {
		return nil, nil, fmt.Errorf("encode dp5m: %w", err)
	}
	dp50, err := track.EncodeVarint(pts.SimplifyDP(track.PreviewPolylineEpsilon50M))
	if err != nil {
		return nil, nil, fmt.Errorf("encode dp50m: %w", err)
	}
	return dp5, dp50, nil
}

// EnqueueBackfillPreviewPolylineIfNeeded enqueues a single backfill job
// when at least one track still lacks a preview polyline. Safe to call on
// every startup; the job itself reschedules until everything is processed.
func EnqueueBackfillPreviewPolylineIfNeeded(ctx context.Context, d *db.DB, s *Submitter) error {
	n, err := d.QueryRO().CountTracksMissingPreviewPolylines(ctx)
	if err != nil {
		return fmt.Errorf("count missing preview polylines: %w", err)
	}
	if n == 0 {
		return nil
	}
	logg.Info(ctx, "preview polyline backfill: enqueueing", "remaining", n)
	return Submit(ctx, s, BackfillPreviewPolylineArgs{}, Params{})
}
