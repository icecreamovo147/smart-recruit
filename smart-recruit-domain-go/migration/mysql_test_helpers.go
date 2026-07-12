//go:build mysql

package migration

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"gorm.io/gorm"
)

func maxMigrationVersion(t *testing.T, db *gorm.DB) int {
	t.Helper()
	runner, err := NewRunner(db, testMigrationsFS, ".")
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	migrations, err := runner.loadMigrations()
	if err != nil {
		t.Fatalf("loadMigrations: %v", err)
	}
	if len(migrations) == 0 {
		t.Fatal("no migrations loaded")
	}
	return migrations[len(migrations)-1].Version
}

// applyMigrationSQLUpTo executes migration SQL files up to targetVersion without
// writing schema_migrations records. Used to seed partial schemas for baseline tests.
func applyMigrationSQLUpTo(t *testing.T, db *gorm.DB, ctx context.Context, targetVersion int) {
	t.Helper()
	runner, err := NewRunner(db, testMigrationsFS, ".")
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	migrations, err := runner.loadMigrations()
	if err != nil {
		t.Fatalf("loadMigrations: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB: %v", err)
	}

	for _, m := range migrations {
		if m.Version > targetVersion {
			break
		}
		execMigrationStatements(t, sqlDB, ctx, m.UpSQL, m.Name)
	}
}

func execMigrationStatements(t *testing.T, sqlDB *sql.DB, ctx context.Context, sqlText, name string) {
	t.Helper()
	for _, stmt := range splitStatements(sqlText) {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" || stmt == "--" {
			continue
		}
		stmt = normalizeMySQLDDL(stmt)
		if _, err := sqlDB.ExecContext(ctx, stmt); err != nil {
			t.Fatalf("exec migration %s stmt %q: %v", name, truncate(stmt, 80), err)
		}
	}
}

func assertAllMigrationsApplied(t *testing.T, runner *Runner, ctx context.Context) {
	t.Helper()
	entries, err := runner.Status(ctx)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	for _, e := range entries {
		if !e.Applied {
			t.Errorf("migration v%d (%s) should be applied", e.Version, e.Name)
		}
	}
}
