package api

import (
	"database/sql"
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/microcosm-cc/bluemonday"
	blackfriday "github.com/russross/blackfriday/v2"
	"jo-m.ch/go/cartomancer/internal/pkg/db"
	"jo-m.ch/go/cartomancer/internal/pkg/logg"
	"jo-m.ch/go/cartomancer/internal/pkg/session"
)

// maxCommentBodyBytes is the maximum allowed size of a comment body in bytes.
const maxCommentBodyBytes = 10000

// commentSanitizer is a bluemonday policy that allows basic inline formatting,
// lists, and hyperlinks in rendered comment HTML.
var commentSanitizer = func() *bluemonday.Policy {
	p := bluemonday.NewPolicy()
	p.AllowElements("p", "br", "em", "strong", "del", "ul", "ol", "li")
	p.AllowAttrs("href", "rel", "target").OnElements("a")
	p.AllowStandardURLs()
	p.RequireNoFollowOnLinks(true)
	return p
}()

// hrefRe matches href="..." attributes in anchor tags for URL rewriting.
var hrefRe = regexp.MustCompile(`(<a\s[^>]*?)href="([^"]*)"`)

// rewriteLinks rewrites all href attributes in anchor tags to route through
// the /leaving interstitial page, and adds target="_blank".
func rewriteLinks(html string) string {
	return hrefRe.ReplaceAllStringFunc(html, func(match string) string {
		parts := hrefRe.FindStringSubmatch(match)
		if len(parts) != 3 {
			return match
		}
		return parts[1] + `href="/leaving?url=` + url.QueryEscape(parts[2]) + `" target="_blank"`
	})
}

// renderCommentBody converts a markdown comment body to sanitized HTML.
// Only bold, italic, strikethrough, lists, and links are preserved.
// Links are rewritten to go through the /leaving interstitial page.
func renderCommentBody(md string) string {
	extensions := blackfriday.CommonExtensions &^ (blackfriday.Tables | blackfriday.FencedCode | blackfriday.Footnotes | blackfriday.HeadingIDs | blackfriday.Titleblock | blackfriday.DefinitionLists)
	raw := blackfriday.Run([]byte(md), blackfriday.WithExtensions(extensions))
	sanitized := strings.TrimSpace(string(commentSanitizer.SanitizeBytes(raw)))
	return rewriteLinks(sanitized)
}

// commentResponse is the JSON representation of a track comment.
type commentResponse struct {
	UUID      string          `json:"uuid"`
	TrackID   string          `json:"trackId"`
	User      userRefResponse `json:"user"`
	Body      string          `json:"body"`
	BodyHTML  string          `json:"bodyHtml"`
	Deleted   bool            `json:"deleted"`
	CreatedAt string          `json:"createdAt"`
	UpdatedAt string          `json:"updatedAt"`
	CanEdit   bool            `json:"canEdit"`
	CanDelete bool            `json:"canDelete"`
}

// commentsListResponse wraps a list of comments.
type commentsListResponse struct {
	Comments []commentResponse `json:"comments"`
}

type createCommentRequest struct {
	Body string `json:"body"`
}

type editCommentRequest struct {
	Body string `json:"body"`
}

// commentResponseFromRow converts a DB row into a commentResponse, computing
// edit/delete permissions based on the viewer and track owner.
func commentResponseFromRow(row db.ListTrackCommentsRow, viewerID string, trackOwnerID string) commentResponse {
	deleted := row.Deleted != 0
	isAuthor := viewerID != "" && row.UserID == viewerID
	isTrackOwner := viewerID != "" && trackOwnerID == viewerID

	canEdit := !deleted && isAuthor
	canDelete := !deleted && (isAuthor || isTrackOwner)

	var bodyHTML string
	if !deleted {
		bodyHTML = renderCommentBody(row.Body)
	}

	return commentResponse{
		UUID:      row.Uuid,
		TrackID:   row.TrackID,
		User:      userRefResponse{UUID: row.UserID, Name: row.UserName},
		Body:      row.Body,
		BodyHTML:  bodyHTML,
		Deleted:   deleted,
		CreatedAt: row.CreatedAt.Format(time.RFC3339),
		UpdatedAt: row.UpdatedAt.Format(time.RFC3339),
		CanEdit:   canEdit,
		CanDelete: canDelete,
	}
}

// handleListTrackComments handles GET /tracks/{uuid}/comments.
// Returns all comments for a track visible to the caller.
//
// Parameters:
//   - uuid: track UUID (path parameter).
//
// Returns:
//   - 200 with commentsListResponse on success.
//   - 404 if the track does not exist or is not visible.
func (sv *server) handleListTrackComments(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	viewer := session.GetUser(ctx)
	trackUUID := chi.URLParam(r, "uuid")

	t, err := sv.d.QueryRO().GetTrackByUUID(ctx, trackUUID)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "track not found")
		return
	}
	if err != nil {
		logg.Error(ctx, "failed to get track", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	var viewerID string
	if viewer != nil {
		viewerID = viewer.Uuid
	}

	if t.Public == 0 && t.UserID != viewerID {
		writeError(w, http.StatusNotFound, "track not found")
		return
	}

	rows, err := sv.d.QueryRO().ListTrackComments(ctx, trackUUID)
	if err != nil {
		logg.Error(ctx, "failed to list comments", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	comments := make([]commentResponse, len(rows))
	for i, row := range rows {
		comments[i] = commentResponseFromRow(row, viewerID, t.UserID)
	}

	writeJSON(w, http.StatusOK, commentsListResponse{Comments: comments})
}

// handleCreateTrackComment handles POST /tracks/{uuid}/comments.
// Authenticated users can comment on their own tracks or any public track.
//
// Parameters:
//   - uuid: track UUID (path parameter).
//   - body: JSON with "body" field.
//
// Returns:
//   - 201 with the created commentResponse.
//   - 400 if the body is empty.
//   - 404 if the track does not exist or is not visible.
func (sv *server) handleCreateTrackComment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := session.MustGetUser(ctx)
	trackUUID := chi.URLParam(r, "uuid")

	var req createCommentRequest
	if err := decodeJSON(r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}
	if req.Body == "" {
		writeError(w, http.StatusBadRequest, "body is required")
		return
	}
	if len(req.Body) > maxCommentBodyBytes {
		writeError(w, http.StatusBadRequest, "body too long")
		return
	}

	now := time.Now().UTC()
	commentUUID := uuid.Must(uuid.NewV7()).String()

	err := sv.d.WithTx(ctx, func(q *db.Queries) error {
		t, txErr := q.GetTrackByUUID(ctx, trackUUID)
		if txErr != nil {
			return txErr
		}
		if t.Public == 0 && t.UserID != user.Uuid {
			return errTrackNotVisible
		}
		return q.CreateTrackComment(ctx, db.CreateTrackCommentParams{
			Uuid:      commentUUID,
			TrackID:   trackUUID,
			UserID:    user.Uuid,
			Body:      req.Body,
			CreatedAt: now,
			UpdatedAt: now,
		})
	})
	if errors.Is(err, sql.ErrNoRows) || errors.Is(err, errTrackNotVisible) {
		writeError(w, http.StatusNotFound, "track not found")
		return
	}
	if err != nil {
		logg.Error(ctx, "failed to create comment", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	resp := commentResponse{
		UUID:      commentUUID,
		TrackID:   trackUUID,
		User:      userRefResponse{UUID: user.Uuid, Name: user.Name},
		Body:      req.Body,
		BodyHTML:  renderCommentBody(req.Body),
		Deleted:   false,
		CreatedAt: now.Format(time.RFC3339),
		UpdatedAt: now.Format(time.RFC3339),
		CanEdit:   true,
		CanDelete: true,
	}
	writeJSON(w, http.StatusCreated, resp)
}

// handleEditTrackComment handles PATCH /tracks/{trackUUID}/comments/{commentUUID}.
// Only the comment author can edit their own comments.
//
// Parameters:
//   - trackUUID: track UUID (path parameter).
//   - commentUUID: comment UUID (path parameter).
//   - body: JSON with "body" field.
//
// Returns:
//   - 204 on success.
//   - 400 if body is empty.
//   - 403 if the comment is deleted or user is not the author.
//   - 404 if the comment does not exist.
func (sv *server) handleEditTrackComment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := session.MustGetUser(ctx)
	commentUUID := chi.URLParam(r, "commentUUID")

	var req editCommentRequest
	if err := decodeJSON(r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}
	if req.Body == "" {
		writeError(w, http.StatusBadRequest, "body is required")
		return
	}
	if len(req.Body) > maxCommentBodyBytes {
		writeError(w, http.StatusBadRequest, "body too long")
		return
	}

	now := time.Now().UTC()

	err := sv.d.WithTx(ctx, func(q *db.Queries) error {
		comment, txErr := q.GetTrackCommentByUUID(ctx, commentUUID)
		if txErr != nil {
			return txErr
		}
		if comment.Deleted != 0 || comment.UserID != user.Uuid {
			return errForbidden
		}

		_, txErr = q.UpdateTrackCommentBody(ctx, db.UpdateTrackCommentBodyParams{
			Body:      req.Body,
			UpdatedAt: now,
			Uuid:      commentUUID,
		})
		return txErr
	})
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "comment not found")
		return
	}
	if errors.Is(err, errForbidden) {
		writeError(w, http.StatusForbidden, "cannot edit this comment")
		return
	}
	if err != nil {
		logg.Error(ctx, "failed to edit comment", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleDeleteTrackComment handles DELETE /tracks/{trackUUID}/comments/{commentUUID}.
// The comment author or the track owner can soft-delete a comment (body cleared, marked deleted).
//
// Parameters:
//   - trackUUID: track UUID (path parameter).
//   - commentUUID: comment UUID (path parameter).
//
// Returns:
//   - 204 on success.
//   - 403 if user lacks permission.
//   - 404 if the comment does not exist.
func (sv *server) handleDeleteTrackComment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := session.MustGetUser(ctx)
	commentUUID := chi.URLParam(r, "commentUUID")

	now := time.Now().UTC()

	err := sv.d.WithTx(ctx, func(q *db.Queries) error {
		comment, txErr := q.GetTrackCommentByUUID(ctx, commentUUID)
		if txErr != nil {
			return txErr
		}
		if comment.Deleted != 0 {
			return errForbidden
		}

		track, txErr := q.GetTrackByUUID(ctx, comment.TrackID)
		if txErr != nil {
			return txErr
		}

		isAuthor := comment.UserID == user.Uuid
		isTrackOwner := track.UserID == user.Uuid
		if !isAuthor && !isTrackOwner {
			return errForbidden
		}

		_, txErr = q.SoftDeleteTrackComment(ctx, db.SoftDeleteTrackCommentParams{
			UpdatedAt: now,
			Uuid:      commentUUID,
		})
		return txErr
	})
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "comment not found")
		return
	}
	if errors.Is(err, errForbidden) {
		writeError(w, http.StatusForbidden, "cannot delete this comment")
		return
	}
	if err != nil {
		logg.Error(ctx, "failed to delete comment", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
