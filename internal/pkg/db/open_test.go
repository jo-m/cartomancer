package db_test

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"jo-m.ch/go/cartomancer/internal/pkg/db"
)

// queryPragma reads a single PRAGMA value from conn as a string.
// Pragmas that have no value (or that the connection cannot read) return "".
func queryPragma(t *testing.T, conn *sql.DB, name string) string {
	t.Helper()
	var v sql.NullString
	err := conn.QueryRowContext(context.Background(), "PRAGMA "+name).Scan(&v)
	require.NoError(t, err, "PRAGMA %s", name)
	return v.String
}

// TestPragmas_DefaultsAppliedToBothConns verifies that every default pragma
// in [db.Open] is actually applied to both the read/write and the read-only
// connection pool. The busy_timeout assertion is the regression test for the
// fix that moved busy_timeout from the unrecognized _busy_timeout DSN key to a
// _pragma entry; modernc.org/sqlite only recognizes a fixed set of _* DSN
// parameters and silently drops the rest.
func TestPragmas_DefaultsAppliedToBothConns(t *testing.T) {
	t.Parallel()

	d := db.GetTestDB(t)
	t.Cleanup(func() { _ = d.Close() })

	// Default pragmas mirror those declared in internal/pkg/db/open.go.
	// Values are what SQLite reports back from "PRAGMA <name>", which is not
	// always the literal we set (e.g. synchronous=NORMAL reads back as "1").
	cases := []struct {
		name string
		want string
	}{
		{"journal_mode", "wal"},
		{"synchronous", "1"},  // NORMAL
		{"temp_store", "2"},   // MEMORY
		{"foreign_keys", "1"}, // true
		{"cache_size", "1000000000"},
		// SQLite caps mmap_size at SQLITE_MAX_MMAP_SIZE (0x7FFF0000) regardless
		// of the requested value; we send 2 GiB and read back 2 GiB - 64 KiB.
		{"mmap_size", "2147418112"},
		{"busy_timeout", "5000"}, // 5s default in db.Open
	}

	for _, conn := range []struct {
		label string
		db    *sql.DB
	}{
		{"rw", d.RW()},
		{"ro", d.RO()},
	} {
		t.Run(conn.label, func(t *testing.T) {
			for _, c := range cases {
				got := strings.ToLower(queryPragma(t, conn.db, c.name))
				require.Equal(t, c.want, got, "pragma %s", c.name)
			}
		})
	}
}
