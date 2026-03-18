// Package trackgroup groups similar tracks for a user by comparing their H3 cell paths.
// Grouping is done hierarchically: a coarse pass at low H3 resolution identifies candidate
// clusters cheaply, then a fine pass at higher resolution refines each cluster.
package trackgroup

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"jo-m.ch/go/detour/internal/pkg/blob"
	"jo-m.ch/go/detour/internal/pkg/db"
	"jo-m.ch/go/detour/internal/pkg/load"
	"jo-m.ch/go/detour/internal/pkg/logg"
	"jo-m.ch/go/detour/internal/pkg/track"
)

const (
	// maxTrackDistanceM is the maximum total distance of a track to be considered for grouping.
	maxTrackDistanceM = 200_000

	// coarseResolution is the H3 resolution for the initial cheap grouping pass.
	coarseResolution = 4
	// fineResolution is the H3 resolution for the refinement pass within coarse clusters.
	fineResolution = 7

	// coarseMatchRatio is the minimum shared-edge ratio for the coarse pass.
	coarseMatchRatio = 0.5
	// fineMatchRatio is the minimum shared-edge ratio for the fine pass.
	fineMatchRatio = 0.75
)

// trackEntry holds a loaded track together with its database UUID.
type trackEntry struct {
	uuid      string
	coarse    *track.Cells
	blobID    int64
	fileInfo  fileInfo
	trackType track.TrackType
}

// fileInfo captures the minimum needed to reload a blob as a TrackSource.
type fileInfo struct {
	format   track.FileFormat
	filename string
}

// GroupUser loads all groupable tracks for the given user, clusters them, and
// replaces any existing track_groups rows for that user with the new results.
// It skips the expensive work if the set of groupable tracks has not changed
// since the last run (tracked via the newest track UUID as a watermark).
func GroupUser(ctx context.Context, d *db.DB, userID string) error {
	latestUUID, err := d.QueryRO().GetLatestTrackUUIDByUser(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		// No groupable tracks at all; clear any stale state.
		return clearGroups(ctx, d, userID)
	}
	if err != nil {
		return fmt.Errorf("getting latest track UUID: %w", err)
	}

	state, err := d.QueryRO().GetTrackGroupState(ctx, userID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("getting group state: %w", err)
	}
	if err == nil && state.LatestTrackUuid == latestUUID {
		return nil
	}

	entries, err := loadTracks(ctx, d, userID)
	if err != nil {
		return fmt.Errorf("loading tracks: %w", err)
	}

	var groups [][]string
	if len(entries) >= 2 {
		groups, err = groupHierarchically(ctx, d, entries)
		if err != nil {
			return fmt.Errorf("grouping: %w", err)
		}
	}

	return replaceGroups(ctx, d, userID, groups, latestUUID)
}

// clearGroups removes all groups and state for a user who has no groupable tracks.
func clearGroups(ctx context.Context, d *db.DB, userID string) error {
	return d.WithTx(ctx, func(q *db.Queries) error {
		if err := q.DeleteTrackGroupsByUser(ctx, userID); err != nil {
			return fmt.Errorf("deleting old groups: %w", err)
		}
		if err := q.DeleteTrackGroupState(ctx, userID); err != nil {
			return fmt.Errorf("deleting group state: %w", err)
		}
		return nil
	})
}

// loadTracks fetches all groupable tracks and converts them to coarse cells.
func loadTracks(ctx context.Context, d *db.DB, userID string) ([]trackEntry, error) {
	rows, err := d.QueryRO().ListGroupableTracksByUser(ctx, db.ListGroupableTracksByUserParams{
		UserID:         userID,
		TotalDistanceM: maxTrackDistanceM,
	})
	if err != nil {
		return nil, err
	}

	entries := make([]trackEntry, 0, len(rows))
	for _, row := range rows {
		b, err := blob.Get(ctx, d.QueryRO(), row.BlobID)
		if err != nil {
			logg.Error(ctx, "Failed to get blob for track, skipping.", "trackUUID", row.Uuid, "err", err)
			continue
		}

		src, err := load.Blob(row.OriginalFilename, bytes.NewReader(b.Content))
		if err != nil {
			logg.Error(ctx, "Failed to parse track blob, skipping.", "trackUUID", row.Uuid, "err", err)
			continue
		}

		coarse, err := track.NewCells(src, coarseResolution)
		if err != nil {
			logg.Error(ctx, "Failed to build coarse cells, skipping.", "trackUUID", row.Uuid, "err", err)
			continue
		}

		entries = append(entries, trackEntry{
			uuid:      row.Uuid,
			coarse:    coarse,
			blobID:    row.BlobID,
			fileInfo:  fileInfo{format: track.FileFormat(row.FileFormat), filename: row.OriginalFilename},
			trackType: track.TrackType(row.TrackType),
		})
	}
	return entries, nil
}

// groupHierarchically runs a coarse pass to find candidate clusters, then refines
// each cluster at higher resolution. Returns groups as slices of track UUIDs.
func groupHierarchically(ctx context.Context, d *db.DB, entries []trackEntry) ([][]string, error) {
	coarseCells := make([]*track.Cells, len(entries))
	for i, e := range entries {
		coarseCells[i] = e.coarse
	}

	coarseResult, err := track.Group(coarseCells, coarseMatchRatio)
	if err != nil {
		return nil, fmt.Errorf("coarse grouping: %w", err)
	}

	var finalGroups [][]string
	for _, coarseGroup := range coarseResult.Groups {
		refined, err := refineGroup(ctx, d, entries, coarseGroup)
		if err != nil {
			return nil, fmt.Errorf("refining group: %w", err)
		}
		finalGroups = append(finalGroups, refined...)
	}
	return finalGroups, nil
}

// refineGroup takes a coarse cluster (set of entry indices) and re-groups them at fine resolution.
func refineGroup(ctx context.Context, d *db.DB, entries []trackEntry, members map[int]struct{}) ([][]string, error) {
	indices := make([]int, 0, len(members))
	for i := range members {
		indices = append(indices, i)
	}

	fineCells := make([]*track.Cells, len(indices))
	for j, idx := range indices {
		e := entries[idx]
		b, err := blob.Get(ctx, d.QueryRO(), e.blobID)
		if err != nil {
			return nil, fmt.Errorf("blob for track %s: %w", e.uuid, err)
		}
		src, err := load.Blob(e.fileInfo.filename, bytes.NewReader(b.Content))
		if err != nil {
			return nil, fmt.Errorf("parse track %s: %w", e.uuid, err)
		}
		cells, err := track.NewCells(src, fineResolution)
		if err != nil {
			return nil, fmt.Errorf("fine cells for track %s: %w", e.uuid, err)
		}
		fineCells[j] = cells
	}

	fineResult, err := track.Group(fineCells, fineMatchRatio)
	if err != nil {
		return nil, fmt.Errorf("fine grouping: %w", err)
	}

	var groups [][]string
	for _, g := range fineResult.Groups {
		uuids := make([]string, 0, len(g))
		for j := range g {
			uuids = append(uuids, entries[indices[j]].uuid)
		}
		groups = append(groups, uuids)
	}
	return groups, nil
}

// replaceGroups deletes all existing groups for the user, inserts new ones,
// and records the watermark so the next call can skip unchanged data.
func replaceGroups(ctx context.Context, d *db.DB, userID string, groups [][]string, latestTrackUUID string) error {
	return d.WithTx(ctx, func(q *db.Queries) error {
		if err := q.DeleteTrackGroupsByUser(ctx, userID); err != nil {
			return fmt.Errorf("deleting old groups: %w", err)
		}

		now := time.Now()
		for _, uuids := range groups {
			groupID, err := uuid.NewV7()
			if err != nil {
				return fmt.Errorf("generating group UUID: %w", err)
			}
			groupUUID := groupID.String()
			if err := q.CreateTrackGroup(ctx, db.CreateTrackGroupParams{
				Uuid:      groupUUID,
				CreatedAt: now,
				UserID:    userID,
			}); err != nil {
				return fmt.Errorf("creating group: %w", err)
			}

			for _, trackUUID := range uuids {
				if err := q.CreateTrackGroupMember(ctx, db.CreateTrackGroupMemberParams{
					GroupID: groupUUID,
					TrackID: trackUUID,
				}); err != nil {
					return fmt.Errorf("adding member: %w", err)
				}
			}
		}

		if err := q.UpsertTrackGroupState(ctx, db.UpsertTrackGroupStateParams{
			UserID:          userID,
			LatestTrackUuid: latestTrackUUID,
			CreatedAt:       now,
		}); err != nil {
			return fmt.Errorf("saving group state: %w", err)
		}

		return nil
	})
}
