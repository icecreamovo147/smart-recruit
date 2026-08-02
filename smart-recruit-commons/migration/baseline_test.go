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
	const baselineVersion = 89

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
		err = runner.Baseline(ctx, baselineVersion)
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

	t.Run("existing migration history adopts baseline without changing data", func(t *testing.T) {
		cleanup := setupTestDB(t, rootSQL, dbName+"_adopt")
		defer cleanup()
		dsn := replaceDBName(baseDSN, dbName+"_adopt")
		db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{TranslateError: true})
		if err != nil {
			t.Fatal(err)
		}
		sqlDB, _ := db.DB()
		defer sqlDB.Close()

		importBaselineSQL(t, sqlDB, ctx)
		archiveRunner, err := NewRunner(db, testMigrationsFS, "archive/pre-baseline-000089")
		if err != nil {
			t.Fatal(err)
		}
		if err := archiveRunner.ensureTable(ctx); err != nil {
			t.Fatalf("create migration history table: %v", err)
		}
		seedArchivedHistory(t, archiveRunner, ctx, baselineVersion-1)

		if _, err := sqlDB.ExecContext(ctx,
			`INSERT INTO users (username, password, account_type, status)
			 VALUES ('baseline-adoption-sentinel', 'not-a-real-password', 'candidate', 'active')`,
		); err != nil {
			t.Fatalf("insert sentinel: %v", err)
		}
		var beforeCount int
		if err := sqlDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&beforeCount); err != nil {
			t.Fatalf("count users before adoption: %v", err)
		}

		runner, err := NewRunner(db, testMigrationsFS, ".")
		if err != nil {
			t.Fatal(err)
		}
		if err := runner.AdoptBaseline(ctx, baselineVersion); err != nil {
			t.Fatalf("AdoptBaseline(%d) should succeed: %v", baselineVersion, err)
		}
		if err := runner.AdoptBaseline(ctx, baselineVersion); err != nil {
			t.Fatalf("repeated AdoptBaseline(%d) should be idempotent: %v", baselineVersion, err)
		}
		if err := runner.Up(ctx); err != nil {
			t.Fatalf("Up after baseline adoption should succeed: %v", err)
		}
		assertAllMigrationsApplied(t, runner, ctx)

		var afterCount int
		if err := sqlDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&afterCount); err != nil {
			t.Fatalf("count users after adoption: %v", err)
		}
		if afterCount != beforeCount {
			t.Fatalf("baseline adoption changed business data: users before=%d after=%d", beforeCount, afterCount)
		}
		var baselineRows int
		if err := sqlDB.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM schema_migrations WHERE version = ?", baselineVersion,
		).Scan(&baselineRows); err != nil {
			t.Fatalf("count baseline records: %v", err)
		}
		if baselineRows != 1 {
			t.Fatalf("baseline record count = %d, want 1", baselineRows)
		}
	})

	t.Run("incomplete migration history rejects adoption", func(t *testing.T) {
		cleanup := setupTestDB(t, rootSQL, dbName+"_incomplete")
		defer cleanup()
		dsn := replaceDBName(baseDSN, dbName+"_incomplete")
		db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{TranslateError: true})
		if err != nil {
			t.Fatal(err)
		}
		sqlDB, _ := db.DB()
		defer sqlDB.Close()

		importBaselineSQL(t, sqlDB, ctx)
		archiveRunner, err := NewRunner(db, testMigrationsFS, "archive/pre-baseline-000089")
		if err != nil {
			t.Fatal(err)
		}
		if err := archiveRunner.ensureTable(ctx); err != nil {
			t.Fatalf("create migration history table: %v", err)
		}
		seedArchivedHistory(t, archiveRunner, ctx, baselineVersion-2)

		runner, err := NewRunner(db, testMigrationsFS, ".")
		if err != nil {
			t.Fatal(err)
		}
		if err := runner.AdoptBaseline(ctx, baselineVersion); err == nil {
			t.Fatal("expected incomplete migration history to reject baseline adoption")
		}
		assertBaselineNotRecorded(t, sqlDB, ctx, baselineVersion)
	})

	t.Run("schema drift rejects adoption", func(t *testing.T) {
		cleanup := setupTestDB(t, rootSQL, dbName+"_drift")
		defer cleanup()
		dsn := replaceDBName(baseDSN, dbName+"_drift")
		db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{TranslateError: true})
		if err != nil {
			t.Fatal(err)
		}
		sqlDB, _ := db.DB()
		defer sqlDB.Close()

		importBaselineSQL(t, sqlDB, ctx)
		archiveRunner, err := NewRunner(db, testMigrationsFS, "archive/pre-baseline-000089")
		if err != nil {
			t.Fatal(err)
		}
		if err := archiveRunner.ensureTable(ctx); err != nil {
			t.Fatalf("create migration history table: %v", err)
		}
		seedArchivedHistory(t, archiveRunner, ctx, baselineVersion-1)
		if _, err := sqlDB.ExecContext(ctx,
			"ALTER TABLE event_outbox ALTER COLUMN producer SET DEFAULT 'drifted-producer'",
		); err != nil {
			t.Fatalf("introduce schema drift: %v", err)
		}

		runner, err := NewRunner(db, testMigrationsFS, ".")
		if err != nil {
			t.Fatal(err)
		}
		if err := runner.AdoptBaseline(ctx, baselineVersion); err == nil {
			t.Fatal("expected schema drift to reject baseline adoption")
		}
		assertBaselineNotRecorded(t, sqlDB, ctx, baselineVersion)
	})
}

func seedArchivedHistory(t *testing.T, runner *Runner, ctx context.Context, throughVersion int) {
	t.Helper()
	archived, err := runner.loadMigrations()
	if err != nil {
		t.Fatalf("load archived migrations: %v", err)
	}
	for _, migration := range archived {
		if migration.Version > throughVersion {
			break
		}
		if err := runner.insertSeedRecord(ctx, nil, migration); err != nil {
			t.Fatalf("seed archived migration %d: %v", migration.Version, err)
		}
	}
}

func assertBaselineNotRecorded(t *testing.T, db *sql.DB, ctx context.Context, version int) {
	t.Helper()
	var count int
	if err := db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM schema_migrations WHERE version = ?", version,
	).Scan(&count); err != nil {
		t.Fatalf("count baseline records: %v", err)
	}
	if count != 0 {
		t.Fatalf("baseline version %d was recorded after rejected adoption", version)
	}
}

// setupTestDB creates a fresh database and returns a cleanup function.
func setupTestDB(t *testing.T, rootSQL *sql.DB, dbName string) func() {
	t.Helper()
	rootSQL.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", dbName))
	if _, err := rootSQL.Exec(fmt.Sprintf(
		"CREATE DATABASE `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci",
		dbName,
	)); err != nil {
		t.Fatalf("create db %s: %v", dbName, err)
	}
	return func() {
		rootSQL.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", dbName))
	}
}

// importDBSQL reads and executes db.sql against the given database.
func importDBSQL(t *testing.T, sqlDB *sql.DB, ctx context.Context) {
	t.Helper()
	importSQLFile(t, sqlDB, ctx, filepath.Join("..", "..", "db.sql"))
}

// importBaselineSQL reads and executes the immutable v89 baseline snapshot.
// Adoption scenarios must use the schema for the version being adopted rather
// than the latest db.sql snapshot, which may include active post-baseline
// migrations.
func importBaselineSQL(t *testing.T, sqlDB *sql.DB, ctx context.Context) {
	t.Helper()
	importSQLFile(t, sqlDB, ctx, filepath.Join("..", "migrations", "000089_schema_baseline.sql"))
}

func importSQLFile(t *testing.T, sqlDB *sql.DB, ctx context.Context, sqlPath string) {
	t.Helper()
	data, err := os.ReadFile(sqlPath)
	if err != nil {
		t.Fatalf("read %s: %v", sqlPath, err)
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
			t.Fatalf("exec %s stmt %q: %v", sqlPath, truncate(stmt, 80), err)
		}
	}
}
