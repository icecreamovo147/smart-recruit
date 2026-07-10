//go:build mysql

package migration

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// TestBaselineScenarios tests the Baseline method against a MySQL database.
//
// Run with: go test -tags=mysql -run TestBaselineScenarios ./migration/
// Requires MYSQL_DSN env var pointing to a MySQL 8.0 instance.
func TestBaselineScenarios(t *testing.T) {
	baseDSN := os.Getenv("MYSQL_DSN")
	if baseDSN == "" {
		t.Skip("MYSQL_DSN not set; skipping Baseline test")
	}

	ctx := context.Background()

	// ── Connect ──────────────────────────────────────────────────────
	rootDB, err := gorm.Open(mysql.Open(baseDSN), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect mysql: %v", err)
	}
	rootSQL, _ := rootDB.DB()
	defer rootSQL.Close()

	dbName := "recruitment_baseline_test"

	t.Run("empty database rejects", func(t *testing.T) {
		cleanup := setupTestDB(t, rootSQL, dbName+"_empty")
		defer cleanup()
		dsn := replaceDBName(baseDSN, dbName+"_empty")
		db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
		if err != nil {
			t.Fatal(err)
		}
		sqlDB, _ := db.DB()
		defer sqlDB.Close()

		runner, err := NewRunner(db, testMigrationsFS, ".")
		if err != nil {
			t.Fatal(err)
		}
		err = runner.Baseline(ctx, 1)
		if err == nil {
			t.Fatal("expected error for empty database, got nil")
		}
		t.Logf("correctly rejected empty db: %v", err)
	})

	t.Run("full snapshot succeeds", func(t *testing.T) {
		cleanup := setupTestDB(t, rootSQL, dbName+"_snapshot")
		defer cleanup()
		dsn := replaceDBName(baseDSN, dbName+"_snapshot")
		db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
		if err != nil {
			t.Fatal(err)
		}
		sqlDB, _ := db.DB()
		defer sqlDB.Close()

		importDBSQL(t, sqlDB, ctx)

		runner, err := NewRunner(db, testMigrationsFS, ".")
		if err != nil {
			t.Fatal(err)
		}
		maxVersion := maxMigrationVersion(t, db)
		if err := runner.Baseline(ctx, maxVersion); err != nil {
			t.Fatalf("Baseline(%d) on full snapshot should succeed: %v", maxVersion, err)
		}
		t.Logf("Baseline(%d) succeeded on full db.sql snapshot", maxVersion)
	})

	t.Run("existing records rejects", func(t *testing.T) {
		cleanup := setupTestDB(t, rootSQL, dbName+"_existing")
		defer cleanup()
		dsn := replaceDBName(baseDSN, dbName+"_existing")
		db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
		if err != nil {
			t.Fatal(err)
		}
		sqlDB, _ := db.DB()
		defer sqlDB.Close()

		importDBSQL(t, sqlDB, ctx)

		runner, err := NewRunner(db, testMigrationsFS, ".")
		if err != nil {
			t.Fatal(err)
		}
		maxVersion := maxMigrationVersion(t, db)
		if err := runner.Baseline(ctx, maxVersion); err != nil {
			t.Fatalf("first Baseline(%d) should succeed: %v", maxVersion, err)
		}
		if err := runner.Baseline(ctx, maxVersion); err == nil {
			t.Fatalf("second Baseline(%d) should be rejected, got nil", maxVersion)
		} else {
			t.Logf("correctly rejected second baseline: %v", err)
		}
	})

	t.Run("full snapshot baseline then up is no-op", func(t *testing.T) {
		cleanup := setupTestDB(t, rootSQL, dbName+"_noop")
		defer cleanup()
		dsn := replaceDBName(baseDSN, dbName+"_noop")
		db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{TranslateError: true})
		if err != nil {
			t.Fatal(err)
		}
		sqlDB, _ := db.DB()
		defer sqlDB.Close()

		importDBSQL(t, sqlDB, ctx)

		runner, err := NewRunner(db, testMigrationsFS, ".")
		if err != nil {
			t.Fatal(err)
		}

		maxVersion := maxMigrationVersion(t, db)
		if err := runner.Baseline(ctx, maxVersion); err != nil {
			t.Fatalf("Baseline(%d) should succeed: %v", maxVersion, err)
		}
		if err := runner.Up(ctx); err != nil {
			t.Fatalf("Up after full baseline should succeed: %v", err)
		}
		assertAllMigrationsApplied(t, runner, ctx)
		t.Logf("Up after Baseline(%d) completed with no pending migrations", maxVersion)
	})

	t.Run("incremental migrations up after partial schema", func(t *testing.T) {
		cleanup := setupTestDB(t, rootSQL, dbName+"_up_after")
		defer cleanup()
		dsn := replaceDBName(baseDSN, dbName+"_up_after")
		db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{TranslateError: true})
		if err != nil {
			t.Fatal(err)
		}
		sqlDB, _ := db.DB()
		defer sqlDB.Close()

		const partialVersion = 21
		applyMigrationSQLUpTo(t, db, ctx, partialVersion)

		runner, err := NewRunner(db, testMigrationsFS, ".")
		if err != nil {
			t.Fatal(err)
		}
		if err := runner.Baseline(ctx, partialVersion); err != nil {
			t.Fatalf("Baseline(%d) should succeed: %v", partialVersion, err)
		}
		if err := runner.Up(ctx); err != nil {
			t.Fatalf("Up after partial baseline should succeed: %v", err)
		}
		assertAllMigrationsApplied(t, runner, ctx)
		t.Logf("Up after Baseline(%d) applied remaining migrations successfully", partialVersion)
	})
}

// setupTestDB creates a fresh database and returns a cleanup function.
func setupTestDB(t *testing.T, rootSQL *sql.DB, dbName string) func() {
	t.Helper()
	rootSQL.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", dbName))
	if _, err := rootSQL.Exec(fmt.Sprintf("CREATE DATABASE `%s`", dbName)); err != nil {
		t.Fatalf("create db %s: %v", dbName, err)
	}
	return func() {
		rootSQL.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", dbName))
	}
}

// importDBSQL reads and executes db.sql against the given database.
func importDBSQL(t *testing.T, sqlDB *sql.DB, ctx context.Context) {
	t.Helper()
	dbSQLPath := filepath.Join("..", "..", "db.sql")
	data, err := os.ReadFile(dbSQLPath)
	if err != nil {
		t.Fatalf("read db.sql: %v", err)
	}
	statements := splitStatements(string(data))
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if isIgnoredDBStatement(stmt) {
			continue
		}
		if _, err := sqlDB.ExecContext(ctx, stmt); err != nil {
			t.Fatalf("exec db.sql stmt %q: %v", truncate(stmt, 80), err)
		}
	}
}
