package stac

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// makeItem builds a minimal STAC Item for testing.
// refTimeRFC3339 is the forecast:reference_datetime (model initialisation time),
// horizon is an ISO 8601 string such as "PT6H".
func makeItem(id, variable, refTimeRFC3339, horizon string, perturbed bool) Item {
	return Item{
		ID: id,
		Properties: ItemProperties{
			ReferenceDatetime: refTimeRFC3339,
			Variable:          variable,
			Horizon:           horizon,
			Perturbed:         perturbed,
		},
		Assets: map[string]Asset{"data": {Href: "https://example.com/" + id}},
	}
}

// TestFetchItemsForVariables_SelectsNewestRun verifies that only items from the
// newest forecast:reference_datetime are returned, regardless of the order in
// which the API returns items, and that variable filtering is applied.
func TestFetchItemsForVariables_SelectsNewestRun(t *testing.T) {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	newRef := base.Add(6 * time.Hour) // 06:00
	oldRef := base                    // 00:00

	// Build items for both runs in mixed order (old first, then new, then old
	// again) to ensure the function is not order-dependent.
	features := []Item{
		makeItem("old-FOO-1", "FOO", oldRef.Format(time.RFC3339), "PT1H", false),
		makeItem("old-FOO-2", "FOO", oldRef.Format(time.RFC3339), "PT2H", false),
		makeItem("new-FOO-1", "FOO", newRef.Format(time.RFC3339), "PT1H", false),
		makeItem("new-FOO-2", "FOO", newRef.Format(time.RFC3339), "PT2H", false),
		makeItem("new-FOO-3", "FOO", newRef.Format(time.RFC3339), "PT3H", false),
		makeItem("old-FOO-3", "FOO", oldRef.Format(time.RFC3339), "PT3H", false),
		// BAR items from the newest run must be excluded by variable filter.
		makeItem("new-BAR-1", "BAR", newRef.Format(time.RFC3339), "PT1H", false),
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ItemCollection{Features: features})
	}))
	t.Cleanup(srv.Close)

	items, refTime, err := FetchItemsForVariables(context.Background(), srv.URL, []string{"FOO"})
	require.NoError(t, err)
	require.Equal(t, newRef, refTime, "should select the newest reference time")
	require.Len(t, items, 3, "only the 3 FOO items from the newest run should be returned")
	for _, item := range items {
		require.Equal(t, "FOO", item.Properties.Variable)
		rt, parseErr := time.Parse(time.RFC3339, item.Properties.ReferenceDatetime)
		require.NoError(t, parseErr)
		require.Equal(t, newRef, rt, "all returned items must belong to the newest run")
	}
}

// TestFetchItemsForVariables_Pagination verifies that the function follows
// pagination links and collects items from all pages before selecting the
// newest run.
func TestFetchItemsForVariables_Pagination(t *testing.T) {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	newRef := base.Add(6 * time.Hour)
	oldRef := base

	// Page 1: only old-run items.
	page1 := ItemCollection{
		Features: []Item{
			makeItem("old-FOO-1", "FOO", oldRef.Format(time.RFC3339), "PT1H", false),
			makeItem("old-FOO-2", "FOO", oldRef.Format(time.RFC3339), "PT2H", false),
		},
	}
	// Page 2: new-run items — the newest reference time lives on this page.
	page2 := ItemCollection{
		Features: []Item{
			makeItem("new-FOO-1", "FOO", newRef.Format(time.RFC3339), "PT1H", false),
			makeItem("new-FOO-2", "FOO", newRef.Format(time.RFC3339), "PT2H", false),
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("page") == "2" {
			_ = json.NewEncoder(w).Encode(page2)
			return
		}
		// Attach a "next" link for the first page.
		p := page1
		p.Links = []Link{{Rel: "next", Href: "http://" + r.Host + r.URL.Path + "?page=2"}}
		_ = json.NewEncoder(w).Encode(p)
	}))
	t.Cleanup(srv.Close)

	items, refTime, err := FetchItemsForVariables(context.Background(), srv.URL, []string{"FOO"})
	require.NoError(t, err)
	require.Equal(t, newRef, refTime)
	require.Len(t, items, 2)
	for _, item := range items {
		rt, _ := time.Parse(time.RFC3339, item.Properties.ReferenceDatetime)
		require.Equal(t, newRef, rt)
	}
}

// TestFetchItemsForVariables_Empty verifies behaviour when the API returns no items.
func TestFetchItemsForVariables_Empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(ItemCollection{})
	}))
	t.Cleanup(srv.Close)

	items, refTime, err := FetchItemsForVariables(context.Background(), srv.URL, []string{"FOO"})
	require.NoError(t, err)
	require.Empty(t, items)
	require.True(t, refTime.IsZero())
}
