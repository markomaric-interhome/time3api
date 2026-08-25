package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

type migration struct {
	version int
	name    string
	sql     string
}

func Migrate(ctx context.Context, db *sql.DB) error {
	currentVersion, err := getCurrentVersion(ctx, db)
	if err != nil {
		return fmt.Errorf("get current database version: %w", err)
	}

	migrations, err := loadMigrations()
	if err != nil {
		return fmt.Errorf("load migrations: %w", err)
	}

	if len(migrations) == 0 {
		return nil
	}

	latestVersion := migrations[len(migrations)-1].version

	if currentVersion > latestVersion {
		return fmt.Errorf(
			"database version %d is newer than latest migration %d",
			currentVersion,
			latestVersion,
		)
	}

	for _, migration := range migrations {
		if migration.version <= currentVersion {
			continue
		}

		if err := applyMigration(ctx, db, migration); err != nil {
			return fmt.Errorf(
				"apply migration %03d_%s: %w",
				migration.version,
				migration.name,
				err,
			)
		}

		currentVersion = migration.version
	}

	return nil
}

func getCurrentVersion(ctx context.Context, db *sql.DB) (int, error) {
	var version int

	err := db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version)
	if err != nil {
		return 0, err
	}

	return version, nil
}

func loadMigrations() ([]migration, error) {
	entries, err := fs.ReadDir(migrationFiles, "migrations")
	if err != nil {
		return nil, err
	}

	migrations := make([]migration, 0, len(entries))

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		version, name, err := parseMigrationFilename(entry.Name())
		if err != nil {
			return nil, err
		}

		content, err := migrationFiles.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", entry.Name(), err)
		}

		migrations = append(migrations, migration{
			version: version,
			name:    name,
			sql:     string(content),
		})
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].version < migrations[j].version
	})

	for i := 1; i < len(migrations); i++ {
		if migrations[i].version == migrations[i-1].version {
			return nil, fmt.Errorf(
				"duplicate migration version %d",
				migrations[i].version,
			)
		}
	}

	return migrations, nil
}

func parseMigrationFilename(filename string) (int, string, error) {
	filename = strings.TrimSuffix(filename, ".sql")
	parts := strings.SplitN(filename, "_", 2)

	if len(parts) != 2 {
		return 0, "", fmt.Errorf(
			"invalid migration filename %q: expected format 001_name.sql",
			filename+".sql",
		)
	}

	version, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, "", fmt.Errorf(
			"invalid migration version in %q: %w",
			filename+".sql",
			err,
		)
	}

	if version <= 0 {
		return 0, "", fmt.Errorf(
			"migration version must be greater than 0 in %q",
			filename+".sql",
		)
	}

	return version, parts[1], nil
}

func applyMigration(ctx context.Context, db *sql.DB, migration migration) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback()
	}()

	if _, err := tx.ExecContext(ctx, migration.sql); err != nil {
		return fmt.Errorf("execute migration: %w", err)
	}

	setVersionSQL := fmt.Sprintf(
		"PRAGMA user_version = %d",
		migration.version,
	)

	if _, err := tx.ExecContext(ctx, setVersionSQL); err != nil {
		return fmt.Errorf("set database version: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration: %w", err)
	}

	return nil
}
