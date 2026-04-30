package api

import (
	"testing"

	"github.com/stretchr/testify/require"
	"jo-m.ch/go/cartomancer/internal/pkg/db"
)

func TestTrackVisibleToUser(t *testing.T) {
	owner := &db.User{Uuid: "owner-uuid"}
	other := &db.User{Uuid: "other-uuid"}

	publicTrack := db.Track{UserID: "owner-uuid", Public: 1}
	privateTrack := db.Track{UserID: "owner-uuid", Public: 0}

	// Public tracks are visible to everyone, including anonymous users.
	require.True(t, trackVisibleToUser(publicTrack, nil))
	require.True(t, trackVisibleToUser(publicTrack, owner))
	require.True(t, trackVisibleToUser(publicTrack, other))

	// Private tracks are only visible to the owner.
	require.False(t, trackVisibleToUser(privateTrack, nil))
	require.True(t, trackVisibleToUser(privateTrack, owner))
	require.False(t, trackVisibleToUser(privateTrack, other))
}
