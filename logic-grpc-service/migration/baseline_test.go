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

		// Import db.sql
		importDBSQL(t, sqlDB, ctx)

		runner, err := NewRunner(db, testMigrationsFS, ".")
		if err != nil {
			t.Fatal(err)
		}
		if err := runner.Baseline(ctx, 20); err != nil {
			t.Fatalf("Baseline(20) on full snapshot should succeed: %v", err)
		}
		t.Log("Baseline(20) succeeded on full db.sql snapshot")
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

		// Import db.sql
		importDBSQL(t, sqlDB, ctx)

		runner, err := NewRunner(db, testMigrationsFS, ".")
		if err != nil {
			t.Fatal(err)
		}
		// First baseline should succeed.
		if err := runner.Baseline(ctx, 20); err != nil {
			t.Fatalf("first Baseline(20) should succeed: %v", err)
		}
		// Second baseline must fail.
		if err := runner.Baseline(ctx, 20); err == nil {
			t.Fatal("second Baseline(20) should be rejected, got nil")
		} else {
			t.Logf("correctly rejected second baseline: %v", err)
		}
	})

	t.Run("full migrations up works after baseline", func(t *testing.T) {
		cleanup := setupTestDB(t, rootSQL, dbName+"_up_after")
		defer cleanup()
		dsn := replaceDBName(baseDSN, dbName+"_up_after")
		db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{TranslateError: true})
		if err != nil {
			t.Fatal(err)
		}
		sqlDB, _ := db.DB()
		defer sqlDB.Close()

		// Import db.sql
		importDBSQL(t, sqlDB, ctx)

		runner, err := NewRunner(db, testMigrationsFS, ".")
		if err != nil {
			t.Fatal(err)
		}

		// Baseline v1-v21 (leaves v22 pending).
		if err := runner.Baseline(ctx, 21); err != nil {
			t.Fatalf("Baseline(21) should succeed: %v", err)
		}

		// Up() should apply v22 only.
		if err := runner.Up(ctx); err != nil {
			t.Fatalf("Up after baseline should succeed: %v", err)
		}

		// Verify v22 is applied.
		entries, err := runner.Status(ctx)
		if err != nil {
			t.Fatal(err)
		}
		v22Found := false
		for _, e := range entries {
			if e.Version == 22 {
				v22Found = true
				if !e.Applied {
					t.Error("v22 should be applied after Up()")
				}
			}
		}
		if !v22Found {
			t.Error("v22 not found in migration status")
		}
		t.Log("Up after baseline completed successfully, v22 applied")
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
		// Skip USE and CREATE DATABASE — we connect directly to the test database.
		if isIgnoredDBStatement(stmt) {
			continue
		}
		if _, err := sqlDB.ExecContext(ctx, stmt); err != nil {
			t.Fatalf("exec db.sql stmt %q: %v", truncate(stmt, 80), err)
		}
	}
}
