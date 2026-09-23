package db

import (
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAccountKindMigrationPreservesLegacyRows(t *testing.T) {
	conn, err := Open(filepath.Join(t.TempDir(), "legacy.db"))
	require.NoError(t, err)
	defer conn.Close()
	_, err = conn.Exec(`CREATE TABLE accounts (id TEXT PRIMARY KEY, account_type TEXT NOT NULL); INSERT INTO accounts VALUES ('a','cash'),('b','margin'),('c','prop'),('d','backtest')`)
	require.NoError(t, err)
	migration, err := fs.ReadFile(migrationsFS, "migrations/000050_account_kind.up.sql")
	require.NoError(t, err)
	_, err = conn.Exec(string(migration))
	require.NoError(t, err)
	rows, err := conn.Query(`SELECT id, account_kind, capabilities FROM accounts ORDER BY id`)
	require.NoError(t, err)
	defer rows.Close()
	var got []string
	for rows.Next() {
		var id, kind, caps string
		require.NoError(t, rows.Scan(&id, &kind, &caps))
		got = append(got, id+":"+kind+":"+caps)
	}
	require.Equal(t, []string{"a:brokerage:[\"cash\"]", "b:brokerage:[\"margin\"]", "c:prop:[]", "d:backtest:[]"}, got)
}

func TestOpenAndMigrate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	conn, err := Open(path)
	require.NoError(t, err)
	defer conn.Close()

	require.NoError(t, Migrate(conn))
	require.NoError(t, conn.Ping())
}

func TestMigrateRecoversDirtyVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dirty.db")
	conn, err := Open(path)
	require.NoError(t, err)
	defer conn.Close()

	require.NoError(t, Migrate(conn))

	_, err = conn.Exec(`UPDATE schema_migrations SET dirty = 1`)
	require.NoError(t, err)

	var dirty int
	require.NoError(t, conn.QueryRow(`SELECT dirty FROM schema_migrations`).Scan(&dirty))
	require.Equal(t, 1, dirty)

	require.NoError(t, Migrate(conn))
	require.NoError(t, conn.QueryRow(`SELECT dirty FROM schema_migrations`).Scan(&dirty))
	require.Equal(t, 0, dirty)
}
