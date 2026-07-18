package migration

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Use os.DirFS for tests since Go embed does not support ../ paths.
var testMigrationsFS = os.DirFS("../migrations")

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	// Use test name as DB identifier to isolate tests.
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	return db
}

func newTestRunner(t *testing.T, db *gorm.DB) *Runner {
	t.Helper()
	// subDir is "" because testMigrationsFS is already rooted at ../migrations.
	r, err := NewRunner(db, testMigrationsFS, ".")
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	return r
}

func TestLoadMigrations(t *testing.T) {
	r := newTestRunner(t, newTestDB(t))
	migrations, err := r.loadMigrations()
	if err != nil {
		t.Fatalf("loadMigrations: %v", err)
	}
	if len(migrations) == 0 {
		t.Fatal("expected at least one migration, got 0")
	}

	// Verify sorted order.
	for i := 1; i < len(migrations); i++ {
		if migrations[i].Version <= migrations[i-1].Version {
			t.Errorf("migrations not sorted: %d <= %d", migrations[i].Version, migrations[i-1].Version)
		}
	}

	// Verify first migration is version 1.
	if migrations[0].Version != 1 {
		t.Errorf("first migration version = %d, want 1", migrations[0].Version)
	}

	// Verify all migrations have non-empty SQL.
	for _, m := range migrations {
		if m.UpSQL == "" {
			t.Errorf("migration %s has empty SQL", m.Name)
		}
		if m.Checksum == "" {
			t.Errorf("migration %s has empty checksum", m.Name)
		}
	}

	t.Logf("loaded %d migrations", len(migrations))
}

func TestEnsureTable(t *testing.T) {
	db := newTestDB(t)
	r := newTestRunner(t, db)

	ctx := context.Background()
	if err := r.ensureTable(ctx); err != nil {
		t.Fatalf("ensureTable: %v", err)
	}

	// Table should exist and be queryable.
	var count int64
	if err := db.Table("schema_migrations").Count(&count).Error; err != nil {
		t.Fatalf("query schema_migrations: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 rows, got %d", count)
	}

	// Calling again should be idempotent.
	if err := r.ensureTable(ctx); err != nil {
		t.Fatalf("ensureTable (second call): %v", err)
	}
}

func TestGetAppliedEmpty(t *testing.T) {
	db := newTestDB(t)
	r := newTestRunner(t, db)
	ctx := context.Background()

	if err := r.ensureTable(ctx); err != nil {
		t.Fatalf("ensureTable: %v", err)
	}

	applied, err := r.getApplied(ctx)
	if err != nil {
		t.Fatalf("getApplied: %v", err)
	}
	if len(applied) != 0 {
		t.Errorf("expected 0 applied, got %d", len(applied))
	}
}

func TestStatusEmpty(t *testing.T) {
	db := newTestDB(t)
	r := newTestRunner(t, db)
	ctx := context.Background()

	entries, err := r.Status(ctx)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected migration entries, got 0")
	}
	for _, e := range entries {
		if e.Applied {
			t.Errorf("migration %d should not be applied", e.Version)
		}
	}
}

func TestSplitStatements(t *testing.T) {
	tests := []struct {
		name string
		sql  string
		want int
	}{
		{
			name: "simple two statements",
			sql:  "CREATE TABLE a (id INT); INSERT INTO a VALUES (1);",
			want: 2,
		},
		{
			name: "with line comments",
			sql:  "-- comment\nCREATE TABLE a (id INT);\n-- another\nINSERT INTO a VALUES (1);",
			want: 2,
		},
		{
			name: "with block comment",
			sql:  "/* comment */\nCREATE TABLE a (id INT);\nINSERT INTO a VALUES (1);",
			want: 2,
		},
		{
			name: "semicolon in string",
			sql:  "INSERT INTO a VALUES ('hello; world');",
			want: 1,
		},
		{
			name: "empty input",
			sql:  "",
			want: 0,
		},
		{
			name: "only comments",
			sql:  "-- just a comment\n/* block */",
			want: 0,
		},
		{
			name: "no trailing semicolon",
			sql:  "CREATE TABLE a (id INT)",
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitStatements(tt.sql)
			if len(got) != tt.want {
				t.Errorf("splitStatements(%q) = %d statements, want %d\ngot: %v", tt.sql, len(got), tt.want, got)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"short", 10, "short"},
		{"this is a long string", 10, "this is a ..."},
		{"exact", 5, "exact"},
	}
	for _, tt := range tests {
		got := truncate(tt.input, tt.maxLen)
		if got != tt.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
		}
	}
}

func TestNormalizeMySQLDDL(t *testing.T) {
	input := "ALTER TABLE users ADD COLUMN IF NOT EXISTS status VARCHAR(32), ADD COLUMN if not exists token_version INT"
	want := "ALTER TABLE users ADD COLUMN status VARCHAR(32), ADD COLUMN token_version INT"
	if got := normalizeMySQLDDL(input); got != want {
		t.Fatalf("normalizeMySQLDDL() = %q, want %q", got, want)
	}
}

func TestMigrationChecksumMatches(t *testing.T) {
	canonical := "canonical"
	if !migrationChecksumMatches(2, canonical, canonical) {
		t.Fatal("canonical checksum should match")
	}
	if !migrationChecksumMatches(2, legacyMigrationChecksums[2], canonical) {
		t.Fatal("known legacy checksum should match")
	}
	if migrationChecksumMatches(2, "unknown", canonical) {
		t.Fatal("unknown checksum must not match")
	}
	if migrationChecksumMatches(1, legacyMigrationChecksums[2], canonical) {
		t.Fatal("legacy checksum must not match a different migration version")
	}
}

func TestRunnerDownNoDownFile(t *testing.T) {
	db := newTestDB(t)
	r := newTestRunner(t, db)
	ctx := context.Background()

	if err := r.ensureTable(ctx); err != nil {
		t.Fatalf("ensureTable: %v", err)
	}

	// Insert a fake applied migration record.
	db.Create(&AppliedMigration{
		Version:  999999,
		Name:     "999999_fake.sql",
		Checksum: "abc",
	})

	// Down should fail because there's no .down.sql file.
	err := r.Down(ctx, 0)
	if err == nil {
		t.Fatal("expected error from Down when .down.sql doesn't exist, got nil")
	}
}

func TestNewRunnerValidation(t *testing.T) {
	// nil db
	_, err := NewRunner(nil, testMigrationsFS, "migrations")
	if err == nil {
		t.Error("expected error for nil db")
	}

	// nil FS
	db := newTestDB(t)
	_, err = NewRunner(db, nil, "migrations")
	if err == nil {
		t.Error("expected error for nil FS")
	}
}

func TestExtractTableNames(t *testing.T) {
	tests := []struct {
		name string
		sql  string
		want []string
	}{
		{
			name: "single create table",
			sql:  "CREATE TABLE `users` (id INT);",
			want: []string{"users"},
		},
		{
			name: "if not exists",
			sql:  "CREATE TABLE IF NOT EXISTS `roles` (id INT);",
			want: []string{"roles"},
		},
		{
			name: "multiple tables",
			sql:  "CREATE TABLE `a` (id INT);\nCREATE TABLE `b` (id INT);",
			want: []string{"a", "b"},
		},
		{
			name: "no backticks",
			sql:  "CREATE TABLE users (id INT);",
			want: []string{"users"},
		},
		{
			name: "no tables",
			sql:  "ALTER TABLE users ADD COLUMN age INT;",
			want: nil,
		},
		{
			name: "duplicate filtered",
			sql:  "CREATE TABLE `a` (id INT);\nCREATE TABLE IF NOT EXISTS `a` (id INT);",
			want: []string{"a"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTableNames(tt.sql)
			if len(got) != len(tt.want) {
				t.Errorf("extractTableNames() = %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("extractTableNames()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestDirtyStateDetection(t *testing.T) {
	db := newTestDB(t)
	r := newTestRunner(t, db)
	ctx := context.Background()

	if err := r.ensureTable(ctx); err != nil {
		t.Fatalf("ensureTable: %v", err)
	}

	// Insert a dirty record using raw SQL to avoid GORM zero-value skipping.
	sqlDB, _ := db.DB()
	_, err2 := sqlDB.ExecContext(ctx,
		"INSERT INTO schema_migrations (version, name, applied_at, checksum, dirty) VALUES (?, ?, ?, ?, ?)",
		1, "000001_init_schema.sql", time.Now(), "abc", true)
	if err2 != nil {
		t.Fatalf("insert dirty record: %v", err2)
	}

	// Up should fail due to dirty state.
	err := r.Up(ctx)
	if err == nil {
		t.Fatal("expected error from Up when dirty migration exists, got nil")
	}
	if !strings.Contains(err.Error(), "dirty") {
		t.Errorf("expected error to mention 'dirty', got: %v", err)
	}
}

func TestChecksumMismatch(t *testing.T) {
	db := newTestDB(t)
	r := newTestRunner(t, db)
	ctx := context.Background()

	if err := r.ensureTable(ctx); err != nil {
		t.Fatalf("ensureTable: %v", err)
	}

	// Load real migrations to get the first one's name.
	migrations, err := r.loadMigrations()
	if err != nil {
		t.Fatalf("loadMigrations: %v", err)
	}
	if len(migrations) == 0 {
		t.Fatal("no migrations loaded")
	}

	// Insert a record with wrong checksum for version 1 using raw SQL.
	sqlDB, _ := db.DB()
	_, insertErr := sqlDB.ExecContext(ctx,
		"INSERT INTO schema_migrations (version, name, applied_at, checksum, dirty) VALUES (?, ?, ?, ?, ?)",
		migrations[0].Version, migrations[0].Name, time.Now(), "wrong_checksum_00000000000000000000000000000000000000000000000000000000", false)
	if insertErr != nil {
		t.Fatalf("insert record: %v", insertErr)
	}

	// Up should fail due to checksum mismatch.
	err = r.Up(ctx)
	if err == nil {
		t.Fatal("expected error from Up when checksum mismatches, got nil")
	}
	if !strings.Contains(err.Error(), "checksum") {
		t.Errorf("expected error to mention 'checksum', got: %v", err)
	}
}
