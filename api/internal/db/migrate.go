package db

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed all:migrations
var migrationsFS embed.FS

//go:embed all:migrations_pg
var migrationsPGFS embed.FS

// Migrate applies schema migrations.
// Optional driver defaults to SQLite; pass DriverPostgres for Postgres.
// SQLite uses the historical migrations/ chain; Postgres uses squashed migrations_pg/.
func Migrate(conn *sql.DB, driver ...string) error {
	d := DriverSQLite
	if len(driver) > 0 && driver[0] != "" {
		d = driver[0]
	}
	switch d {
	case DriverSQLite:
		dbPath, err := sqliteMainFile(conn)
		if err != nil {
			return err
		}
		return withMigrateLock(dbPath, func() error {
			return migrateSQLite(conn)
		})
	case DriverPostgres:
		return migratePostgres(conn)
	default:
		return fmt.Errorf("migrate: unsupported driver %q", d)
	}
}

func migrateSQLite(conn *sql.DB) error {
	src, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return err
	}
	drv, err := sqlite.WithInstance(conn, &sqlite.Config{})
	if err != nil {
		return err
	}

	// air / crash mid-migrate leaves dirty=1 even when DDL mostly applied.
	// Clear it before Up so startup can re-apply idempotent migrations.
	// This rewinds one version, so every up migration must be safe to re-run
	// against a database that already has it (CREATE TABLE/INDEX IF NOT EXISTS).
	if ver, dirty, vErr := drv.Version(); vErr == nil && dirty {
		if ver == 50 {
			var count int
			if err := conn.QueryRow("SELECT count(*) FROM pragma_table_info('accounts') WHERE name IN ('account_kind', 'capabilities')").Scan(&count); err != nil {
				return err
			}
			if count == 1 {
				var existing string
				if err := conn.QueryRow("SELECT name FROM pragma_table_info('accounts') WHERE name IN ('account_kind', 'capabilities')").Scan(&existing); err != nil {
					return err
				}
				stmt := `ALTER TABLE accounts ADD COLUMN capabilities TEXT NOT NULL DEFAULT '["cash"]'`
				if existing == "capabilities" {
					stmt = "ALTER TABLE accounts ADD COLUMN account_kind TEXT NOT NULL DEFAULT 'brokerage'"
				}
				if _, err := conn.Exec(stmt); err != nil {
					return err
				}
				if _, err := conn.Exec(`UPDATE accounts SET account_kind = CASE WHEN account_type IN ('prop', 'backtest', 'paper') THEN account_type ELSE 'brokerage' END, capabilities = CASE account_type WHEN 'margin' THEN '["margin"]' WHEN 'prop' THEN '[]' WHEN 'backtest' THEN '[]' WHEN 'paper' THEN '[]' ELSE '["cash"]' END`); err != nil {
					return err
				}
				count = 2
			}
			if count == 2 {
				// Finish a migration interrupted after both ALTERs but before its backfill.
				if _, err := conn.Exec(`UPDATE accounts SET account_kind = CASE WHEN account_type IN ('prop','backtest','paper') THEN account_type ELSE account_kind END, capabilities = CASE account_type WHEN 'margin' THEN '["margin"]' WHEN 'prop' THEN '[]' WHEN 'backtest' THEN '[]' WHEN 'paper' THEN '[]' ELSE capabilities END WHERE (account_type = 'margin' AND capabilities = '["cash"]') OR (account_type IN ('prop','backtest','paper') AND account_kind = 'brokerage')`); err != nil {
					return err
				}
				if err := drv.SetVersion(ver, false); err != nil {
					return err
				}
				dirty = false
			}
		}
		if ver == 51 {
			var count int
			if err := conn.QueryRow("SELECT count(*) FROM pragma_table_info('trades') WHERE name = 'accounting_warning'").Scan(&count); err != nil {
				return err
			}
			if count == 1 {
				if err := drv.SetVersion(ver, false); err != nil {
					return err
				}
				dirty = false
			}
		}
		if dirty {
			prev := max(ver-1, 0)
			if err := drv.SetVersion(prev, false); err != nil {
				return fmt.Errorf("dirty migration at version %d: could not reset: %w", ver, err)
			}
		}
	}

	m, err := migrate.NewWithInstance("iofs", src, "sqlite", drv)
	if err != nil {
		return err
	}
	// Do not m.Close(): WithInstance owns our *sql.DB and Close would shut it down.

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

func migratePostgres(conn *sql.DB) error {
	src, err := iofs.New(migrationsPGFS, "migrations_pg")
	if err != nil {
		return err
	}
	drv, err := postgres.WithInstance(conn, &postgres.Config{})
	if err != nil {
		return err
	}
	m, err := migrate.NewWithInstance("iofs", src, "postgres", drv)
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

func sqliteMainFile(conn *sql.DB) (string, error) {
	var seq int
	var name, file string
	err := conn.QueryRow(`SELECT seq, name, file FROM pragma_database_list WHERE name = 'main'`).Scan(&seq, &name, &file)
	if err != nil {
		return "", err
	}
	return file, nil
}

func withMigrateLock(dbPath string, fn func() error) error {
	if dbPath == "" || dbPath == ":memory:" || strings.HasPrefix(dbPath, "file::memory:") {
		return fn()
	}
	lockPath := dbPath + ".migrate.lock"
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return fmt.Errorf("migrate lock: %w", err)
	}
	defer f.Close()
	if err := lockMigrateFile(f); err != nil {
		return err
	}
	defer func() { _ = unlockMigrateFile(f) }()
	return fn()
}
