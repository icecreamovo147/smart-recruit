// Package migration provides a versioned database migration runner.
//
// It reads SQL files from an fs.FS (typically an embed.FS passed from main),
// tracks applied versions in a `schema_migrations` table, and supports
// up/down/status operations. A MySQL advisory lock prevents concurrent
// migration execution across instances.
package migration

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"logic-grpc-service/pkg/logger"
)

const (
	lockName    = "smart_recruit_migration"
	lockTimeout = 60 // seconds
)

// filePattern matches migration filenames like "000001_init_schema.sql".
var filePattern = regexp.MustCompile(`^(\d+)_.+\.sql$`)

// createTablePattern matches CREATE TABLE statements to extract table names.
var createTablePattern = regexp.MustCompile(`(?i)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?` + "`?" + `(\w+)` + "`?")

// MySQL 8.0 does not accept ADD COLUMN IF NOT EXISTS, although some compatible
// databases do. Keep historical migration files unchanged for checksum
// stability and normalize the clause only at execution time.
var addColumnIfNotExistsPattern = regexp.MustCompile(`(?i)\bADD\s+COLUMN\s+IF\s+NOT\s+EXISTS\b`)

// legacyMigrationChecksums contains only the checksums shipped by the
// production-readiness commit that rewrote already-published migrations.
// Accepting these exact values lets those databases move back to the canonical
// migration history without weakening checksum validation for other changes.
var legacyMigrationChecksums = map[int]string{
	2:  "a7fbc6258825a439855aa19d62d304207c50fbb37acd86bdcd0f884e310d266c",
	3:  "c8ff0c30a98122920c7e0f1266d1fbd990f0ea85712c901780bb9a1417950a45",
	5:  "4ec291ccfc65e03bb6daa107b6fac269eaa693832072c5402725b8b81a2fb424",
	6:  "cee95e501c5d611364222efcfb314a304b4e491eaf6c3f35d4a187854ff1fb92",
	12: "28caed7cbb9a6a0d49a687a9328b20acad2924579e03c34c96bc09159f760cae",
	14: "5e8a85b71db7971a4eeaf4d6b6c006a90ce83cc1f2fdb033ceaeeae296a6540d",
	17: "d2f52cfed883ad77ec1e1b9e5d32cee4255ff087525b6ae6b55bd0e489e0fcc3",
	22: "7631c39e1df666e3bd99109d298a8c4c207efb54f3826d099f21d9d15c027395",
}

// Migration represents a single migration file.
type Migration struct {
	Version  int    // Parsed from filename prefix, e.g. 000001 → 1
	Name     string // Original filename without directory
	UpSQL    string // SQL content for the UP direction
	Checksum string // SHA256 hex of the SQL content
}

// AppliedMigration is a row in the schema_migrations tracking table.
type AppliedMigration struct {
	Version   int       `gorm:"column:version;primaryKey"`
	Name      string    `gorm:"column:name;type:varchar(255);not null"`
	AppliedAt time.Time `gorm:"column:applied_at;not null;autoCreateTime"`
	Checksum  string    `gorm:"column:checksum;type:varchar(64);not null"`
	Dirty     bool      `gorm:"column:dirty;not null;default:false"`
}

// TableName overrides GORM's default table name.
func (AppliedMigration) TableName() string { return "schema_migrations" }

// StatusEntry describes the state of a single migration for reporting.
type StatusEntry struct {
	Version   int
	Name      string
	Applied   bool
	Dirty     bool
	AppliedAt *time.Time
	Checksum  string
}

// Runner executes versioned SQL migrations against a MySQL database.
type Runner struct {
	db     *gorm.DB
	migrFS fs.FS  // filesystem containing migration SQL files
	subDir string // subdirectory inside migrFS, e.g. "migrations"
}

// NewRunner creates a migration runner.
// migrFS is the filesystem containing migration files (e.g. an embed.FS from main).
// subDir is the directory inside migrFS where .sql files live (e.g. "migrations").
func NewRunner(db *gorm.DB, migrFS fs.FS, subDir string) (*Runner, error) {
	if db == nil {
		return nil, fmt.Errorf("migration: db must not be nil")
	}
	if migrFS == nil {
		return nil, fmt.Errorf("migration: migrFS must not be nil")
	}
	return &Runner{db: db, migrFS: migrFS, subDir: subDir}, nil
}

// loadMigrations scans the FS and returns migrations sorted by version.
func (r *Runner) loadMigrations() ([]Migration, error) {
	entries, err := fs.ReadDir(r.migrFS, r.subDir)
	if err != nil {
		return nil, fmt.Errorf("migration: read dir %q: %w", r.subDir, err)
	}

	prefix := r.subDir
	if prefix == "." || prefix == "" {
		prefix = ""
	} else {
		prefix += "/"
	}

	var migrations []Migration
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		matches := filePattern.FindStringSubmatch(e.Name())
		if matches == nil {
			continue
		}
		// Skip .down.sql files — only load .sql (up) files.
		if strings.HasSuffix(e.Name(), ".down.sql") {
			continue
		}
		version, err := strconv.Atoi(matches[1])
		if err != nil {
			continue
		}
		data, err := fs.ReadFile(r.migrFS, prefix+e.Name())
		if err != nil {
			return nil, fmt.Errorf("migration: read %q: %w", e.Name(), err)
		}
		checksum := fmt.Sprintf("%x", sha256.Sum256(data))

		migrations = append(migrations, Migration{
			Version:  version,
			Name:     e.Name(),
			UpSQL:    string(data),
			Checksum: checksum,
		})
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})
	return migrations, nil
}

// ensureTable creates the schema_migrations tracking table if it does not exist.
func (r *Runner) ensureTable(ctx context.Context) error {
	return r.db.WithContext(ctx).AutoMigrate(&AppliedMigration{})
}

// getApplied returns a map of version → AppliedMigration for all applied migrations.
func (r *Runner) getApplied(ctx context.Context) (map[int]AppliedMigration, error) {
	var rows []AppliedMigration
	if err := r.db.WithContext(ctx).Order("version ASC").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("migration: query applied: %w", err)
	}
	result := make(map[int]AppliedMigration, len(rows))
	for _, row := range rows {
		result[row.Version] = row
	}
	return result, nil
}

// acquireLock obtains a MySQL advisory lock on a pinned connection.
// Returns the underlying *sql.Conn (caller must close) and a release function.
// For non-MySQL databases (e.g. sqlite in tests), it returns nil conn and no-op release.
func (r *Runner) acquireLock(ctx context.Context) (*sql.Conn, func(), error) {
	rawDB, err := r.db.DB()
	if err != nil {
		return nil, nil, fmt.Errorf("migration: get sql.DB: %w", err)
	}

	conn, err := rawDB.Conn(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("migration: pin connection: %w", err)
	}

	var result sql.NullInt64
	err = conn.QueryRowContext(ctx,
		"SELECT CASE WHEN COALESCE(GET_LOCK(?, ?), 0) = 1 THEN 1 ELSE 0 END",
		lockName, lockTimeout,
	).Scan(&result)
	if err != nil {
		// Non-MySQL database (e.g. sqlite) — skip locking, return conn for reuse.
		if r.db.Dialector.Name() == "sqlite" {
			return conn, func() { conn.Close() }, nil
		}
		conn.Close()
		return nil, nil, fmt.Errorf("migration: acquire lock: %w", err)
	}
	if !result.Valid || result.Int64 != 1 {
		conn.Close()
		return nil, nil, fmt.Errorf("migration: could not acquire lock %q (timeout %ds)", lockName, lockTimeout)
	}

	release := func() {
		_, _ = conn.ExecContext(context.Background(), "SELECT RELEASE_LOCK(?)", lockName)
		conn.Close()
	}
	return conn, release, nil
}

// extractTableNames parses CREATE TABLE statements from SQL and returns table names.
func extractTableNames(sqlStr string) []string {
	matches := createTablePattern.FindAllStringSubmatch(sqlStr, -1)
	seen := make(map[string]bool)
	var names []string
	for _, m := range matches {
		name := strings.ToLower(m[1])
		if !seen[name] {
			seen[name] = true
			names = append(names, name)
		}
	}
	return names
}

// tablesExist checks which of the given tables exist in the current database.
func (r *Runner) tablesExist(ctx context.Context, conn *sql.Conn, tables []string) (map[string]bool, error) {
	if len(tables) == 0 {
		return map[string]bool{}, nil
	}

	result := make(map[string]bool, len(tables))
	for _, t := range tables {
		result[t] = false
	}

	// Build query with placeholders for table names.
	placeholders := make([]string, len(tables))
	args := make([]interface{}, len(tables))
	for i, t := range tables {
		placeholders[i] = "?"
		args[i] = t
	}

	query := fmt.Sprintf(
		"SELECT LOWER(table_name) FROM information_schema.tables WHERE table_schema = DATABASE() AND LOWER(table_name) IN (%s)",
		strings.Join(placeholders, ","),
	)

	var rows *sql.Rows
	var err error
	if conn != nil {
		rows, err = conn.QueryContext(ctx, query, args...)
	} else {
		rawDB, dbErr := r.db.DB()
		if dbErr != nil {
			return nil, dbErr
		}
		rows, err = rawDB.QueryContext(ctx, query, args...)
	}
	if err != nil {
		return nil, fmt.Errorf("migration: check tables: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		result[name] = true
	}
	return result, rows.Err()
}

// columnExists checks whether a specific column exists in a table.
func (r *Runner) columnExists(ctx context.Context, conn *sql.Conn, table, column string) (bool, error) {
	var rows *sql.Rows
	var err error
	query := "SELECT 1 FROM information_schema.columns WHERE table_schema = DATABASE() AND LOWER(table_name) = ? AND LOWER(column_name) = ?"
	if conn != nil {
		rows, err = conn.QueryContext(ctx, query, table, column)
	} else {
		rawDB, dbErr := r.db.DB()
		if dbErr != nil {
			return false, dbErr
		}
		rows, err = rawDB.QueryContext(ctx, query, table, column)
	}
	if err != nil {
		return false, fmt.Errorf("migration: check column %s.%s: %w", table, column, err)
	}
	defer rows.Close()
	return rows.Next(), rows.Err()
}

// seedFromExistingTables checks if the tables created by each migration already
// exist (e.g. imported via db.sql) and marks those specific migrations as applied.
// Only migrations whose ALL created tables exist are seeded.
func (r *Runner) seedFromExistingTables(ctx context.Context, conn *sql.Conn, migrations []Migration) error {
	applied, err := r.getApplied(ctx)
	if err != nil {
		return err
	}
	if len(applied) > 0 {
		// Already has migration records, no need to seed.
		return nil
	}

	// Collect all table names referenced across all migrations.
	allTables := make(map[string]bool)
	migrationTables := make([][]string, len(migrations))
	for i, m := range migrations {
		tables := extractTableNames(m.UpSQL)
		migrationTables[i] = tables
		for _, t := range tables {
			allTables[t] = true
		}
	}

	if len(allTables) == 0 {
		return nil
	}

	// Check which tables exist in one query.
	tableList := make([]string, 0, len(allTables))
	for t := range allTables {
		tableList = append(tableList, t)
	}
	existing, err := r.tablesExist(ctx, conn, tableList)
	if err != nil {
		// Non-MySQL (sqlite in tests) — skip seeding.
		return nil
	}

	totalExisting := 0
	for _, e := range existing {
		if e {
			totalExisting++
		}
	}
	if totalExisting <= 1 {
		// Only schema_migrations (or nothing) — fresh database.
		return nil
	}

	log := logger.L()
	seeded := 0
	for i, m := range migrations {
		tables := migrationTables[i]
		if len(tables) == 0 {
			// Migration creates no tables (e.g., ALTER only, seed data).
			// Skip — only pure CREATE TABLE migrations are auto-seeded.
			// Use --migrate-baseline instead for databases initialized via db.sql.
			continue
		}

		// Check if ALL tables created by this migration exist.
		allExist := true
		for _, t := range tables {
			if !existing[t] {
				allExist = false
				break
			}
		}
		if !allExist {
			continue
		}

		log.Info("migration: seeding (tables exist from db.sql)",
			zap.Int("version", m.Version), zap.String("name", m.Name))
		if err := r.insertSeedRecord(ctx, conn, m); err != nil {
			return err
		}
		seeded++
	}

	if seeded > 0 {
		log.Info("migration: seeded history from existing tables",
			zap.Int("seeded", seeded), zap.Int("total_migrations", len(migrations)))
	}
	return nil
}

// insertSeedRecord inserts a migration record, ignoring duplicate key errors.
func (r *Runner) insertSeedRecord(ctx context.Context, conn *sql.Conn, m Migration) error {
	record := AppliedMigration{
		Version:   m.Version,
		Name:      m.Name,
		AppliedAt: time.Now(),
		Checksum:  m.Checksum,
		Dirty:     false,
	}
	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		if !strings.Contains(err.Error(), "Duplicate") {
			return fmt.Errorf("migration: seed record for %s: %w", m.Name, err)
		}
	}
	return nil
}

// Baseline marks migrations v1–targetVersion as applied without executing them.
// Use this when the database was initialized via db.sql (a full schema snapshot)
// and you want to skip the corresponding migrations while still having an
// accurate schema_migrations history. Subsequent Up() calls will only apply
// versions above targetVersion.
// targetVersion must be >= 1 and <= the highest available migration version.
func (r *Runner) Baseline(ctx context.Context, targetVersion int) error {
	if r.db.Dialector.Name() != "mysql" {
		return fmt.Errorf("migration: baseline requires a MySQL database (got %q)", r.db.Dialector.Name())
	}
	if err := r.ensureTable(ctx); err != nil {
		return fmt.Errorf("migration: ensure table: %w", err)
	}

	conn, release, err := r.acquireLock(ctx)
	if err != nil {
		return err
	}
	defer release()

	// Refuse if any records already exist.
	applied, err := r.getApplied(ctx)
	if err != nil {
		return err
	}
	if len(applied) > 0 {
		return fmt.Errorf("migration: baseline refused: schema_migrations already has %d record(s)", len(applied))
	}

	migrations, err := r.loadMigrations()
	if err != nil {
		return err
	}
	if len(migrations) == 0 {
		return fmt.Errorf("migration: no migration files found")
	}

	maxVersion := migrations[len(migrations)-1].Version
	if targetVersion < 1 || targetVersion > maxVersion {
		return fmt.Errorf("migration: baseline target version %d out of range [1, %d]", targetVersion, maxVersion)
	}

	log := logger.L()

	// ── Collect all table names created by v1-targetVersion ──────────
	var allTableNames []string
	for _, m := range migrations {
		if m.Version > targetVersion {
			break
		}
		tables := extractTableNames(m.UpSQL)
		allTableNames = append(allTableNames, tables...)
	}

	// ── Verify all those tables exist ───────────────────────────────
	if len(allTableNames) > 0 {
		existing, err := r.tablesExist(ctx, conn, allTableNames)
		if err != nil {
			return fmt.Errorf("migration: baseline verification: %w", err)
		}
		for _, t := range allTableNames {
			if !existing[t] {
				return fmt.Errorf("migration: baseline verification failed: table %q not found (did you import db.sql?)", t)
			}
		}
	}

	// ── Verify critical columns added by ALTER migrations ──────────
	criticalColumns := []struct {
		table  string
		column string
	}{
		{"users", "account_type"},
		{"users", "status"},
		{"users", "token_version"},
		{"applications", "status_key"},
		{"notifications", "receiver_account_type"},
	}
	for _, cc := range criticalColumns {
		exists, err := r.columnExists(ctx, conn, cc.table, cc.column)
		if err != nil {
			return fmt.Errorf("migration: baseline verification: %w", err)
		}
		if !exists {
			return fmt.Errorf("migration: baseline verification failed: column %q not found in table %q (did you import db.sql?)",
				cc.column, cc.table)
		}
	}

	// ── Verify key Offer, interview, collaboration tables exist ────
	offerTables := []string{"offers", "offer_events", "interview_schedules",
		"interview_feedback", "candidate_notes", "candidate_tags",
		"candidate_tag_assignments", "follow_up_tasks"}
	existing, err := r.tablesExist(ctx, conn, offerTables)
	if err != nil {
		return fmt.Errorf("migration: baseline verification: %w", err)
	}
	for _, t := range offerTables {
		if !existing[t] {
			return fmt.Errorf("migration: baseline verification failed: table %q not found (did you import db.sql?)", t)
		}
	}

	seeded := 0
	for _, m := range migrations {
		if m.Version > targetVersion {
			break
		}
		if err := r.insertSeedRecord(ctx, conn, m); err != nil {
			return fmt.Errorf("migration: baseline seed %s: %w", m.Name, err)
		}
		seeded++
	}

	log.Info("migration: baseline completed",
		zap.Int("target_version", targetVersion),
		zap.Int("seeded", seeded))
	return nil
}

// Up applies all pending migrations in version order.
func (r *Runner) Up(ctx context.Context) error {
	if err := r.ensureTable(ctx); err != nil {
		return fmt.Errorf("migration: ensure table: %w", err)
	}

	conn, release, err := r.acquireLock(ctx)
	if err != nil {
		return err
	}
	defer release()

	migrations, err := r.loadMigrations()
	if err != nil {
		return err
	}

	applied, err := r.getApplied(ctx)
	if err != nil {
		return err
	}

	// Check for dirty (failed) migrations.
	for v, a := range applied {
		if a.Dirty {
			return fmt.Errorf("migration: version %d (%s) is marked dirty (previous execution failed). "+
				"Manual intervention required: fix the issue, then either repair the database and "+
				"DELETE FROM schema_migrations WHERE version = %d, or use --migrate-down to roll back",
				v, a.Name, v)
		}
	}

	// Checksum verification for applied migrations.
	log := logger.L()
	for _, m := range migrations {
		if a, ok := applied[m.Version]; ok {
			if !migrationChecksumMatches(m.Version, a.Checksum, m.Checksum) {
				return fmt.Errorf("migration: checksum mismatch for version %d (%s). "+
					"Applied checksum: %s, file checksum: %s. "+
					"Do not modify already-applied migration files",
					m.Version, m.Name, a.Checksum[:12], m.Checksum[:12])
			}
		}
	}

	pending := 0
	for _, m := range migrations {
		if _, ok := applied[m.Version]; ok {
			continue
		}
		pending++
		log.Info("migration: applying", zap.Int("version", m.Version), zap.String("name", m.Name))

		// Write dirty record BEFORE executing SQL.
		dirtyRecord := AppliedMigration{
			Version:   m.Version,
			Name:      m.Name,
			AppliedAt: time.Now(),
			Checksum:  m.Checksum,
			Dirty:     true,
		}
		if err := r.db.WithContext(ctx).Create(&dirtyRecord).Error; err != nil {
			return fmt.Errorf("migration: create dirty record %s: %w", m.Name, err)
		}

		// Execute SQL on the pinned connection.
		if err := r.execSQLConn(ctx, conn, m.UpSQL); err != nil {
			// Leave dirty record in place — indicates failed migration.
			return fmt.Errorf("migration: apply %s: %w\n  → version %d marked as dirty, manual fix required",
				m.Name, err, m.Version)
		}

		// Mark as clean (success).
		if err := r.db.WithContext(ctx).
			Model(&AppliedMigration{}).
			Where("version = ?", m.Version).
			Update("dirty", false).Error; err != nil {
			return fmt.Errorf("migration: clear dirty flag %s: %w", m.Name, err)
		}

		log.Info("migration: applied successfully", zap.Int("version", m.Version), zap.String("name", m.Name))
	}

	if pending == 0 {
		log.Info("migration: all migrations already applied", zap.Int("total", len(migrations)))
	} else {
		log.Info("migration: completed", zap.Int("applied", pending), zap.Int("total", len(migrations)))
	}
	return nil
}

// Down rolls back migrations down to (but not including) the target version.
// If targetVersion is 0, all migrations are rolled back.
func (r *Runner) Down(ctx context.Context, targetVersion int) error {
	if err := r.ensureTable(ctx); err != nil {
		return fmt.Errorf("migration: ensure table: %w", err)
	}

	conn, release, err := r.acquireLock(ctx)
	if err != nil {
		return err
	}
	defer release()

	applied, err := r.getApplied(ctx)
	if err != nil {
		return err
	}

	var versions []int
	for v := range applied {
		if v > targetVersion {
			versions = append(versions, v)
		}
	}
	sort.Sort(sort.Reverse(sort.IntSlice(versions)))

	if len(versions) == 0 {
		logger.L().Info("migration: nothing to roll back", zap.Int("target", targetVersion))
		return nil
	}

	log := logger.L()
	for _, v := range versions {
		m := applied[v]
		downSQL, err := r.loadDownSQL(m.Name)
		if err != nil {
			return fmt.Errorf("migration: cannot roll back %s: %w", m.Name, err)
		}

		log.Info("migration: rolling back", zap.Int("version", v), zap.String("name", m.Name))
		if err := r.execSQLConn(ctx, conn, downSQL); err != nil {
			return fmt.Errorf("migration: rollback %s: %w", m.Name, err)
		}

		if err := r.db.WithContext(ctx).Delete(&AppliedMigration{}, "version = ?", v).Error; err != nil {
			return fmt.Errorf("migration: delete record %s: %w", m.Name, err)
		}
		log.Info("migration: rolled back", zap.Int("version", v), zap.String("name", m.Name))
	}

	log.Info("migration: rollback completed", zap.Int("rolled_back", len(versions)), zap.Int("target", targetVersion))
	return nil
}

// loadDownSQL reads the down migration SQL for a given up migration filename.
func (r *Runner) loadDownSQL(upName string) (string, error) {
	prefix := r.subDir
	if prefix == "." || prefix == "" {
		prefix = ""
	} else {
		prefix += "/"
	}
	downName := strings.TrimSuffix(upName, ".sql") + ".down.sql"
	data, err := fs.ReadFile(r.migrFS, prefix+downName)
	if err != nil {
		return "", fmt.Errorf("no down migration file found (%s): %w", downName, err)
	}
	return string(data), nil
}

// Status returns the status of all known migrations.
func (r *Runner) Status(ctx context.Context) ([]StatusEntry, error) {
	if err := r.ensureTable(ctx); err != nil {
		return nil, fmt.Errorf("migration: ensure table: %w", err)
	}

	migrations, err := r.loadMigrations()
	if err != nil {
		return nil, err
	}

	applied, err := r.getApplied(ctx)
	if err != nil {
		return nil, err
	}

	var entries []StatusEntry
	for _, m := range migrations {
		entry := StatusEntry{
			Version:  m.Version,
			Name:     m.Name,
			Checksum: m.Checksum,
		}
		if a, ok := applied[m.Version]; ok {
			entry.Applied = true
			entry.Dirty = a.Dirty
			t := a.AppliedAt
			entry.AppliedAt = &t
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

// execSQLConn executes raw SQL statements on a pinned connection.
func (r *Runner) execSQLConn(ctx context.Context, conn *sql.Conn, sqlStr string) error {
	statements := splitStatements(sqlStr)
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" || stmt == "--" {
			continue
		}
		if r.db.Dialector.Name() == "mysql" {
			stmt = normalizeMySQLDDL(stmt)
		}
		if _, err := conn.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("exec %q: %w", truncate(stmt, 80), err)
		}
	}
	return nil
}

func migrationChecksumMatches(version int, applied, canonical string) bool {
	if applied == canonical {
		return true
	}
	legacy, ok := legacyMigrationChecksums[version]
	return ok && applied == legacy
}

func normalizeMySQLDDL(stmt string) string {
	return addColumnIfNotExistsPattern.ReplaceAllString(stmt, "ADD COLUMN")
}

// splitStatements splits SQL text into individual statements on semicolons,
// respecting quoted strings and comments.
func splitStatements(sqlStr string) []string {
	var statements []string
	var current strings.Builder
	inSingleQuote := false
	inDoubleQuote := false
	inLineComment := false
	inBlockComment := false

	for i := 0; i < len(sqlStr); i++ {
		ch := sqlStr[i]

		if inLineComment {
			current.WriteByte(ch)
			if ch == '\n' {
				inLineComment = false
			}
			continue
		}
		if inBlockComment {
			current.WriteByte(ch)
			if ch == '*' && i+1 < len(sqlStr) && sqlStr[i+1] == '/' {
				current.WriteByte(sqlStr[i+1])
				i++
				inBlockComment = false
			}
			continue
		}
		if inSingleQuote {
			current.WriteByte(ch)
			if ch == '\'' {
				inSingleQuote = false
			}
			continue
		}
		if inDoubleQuote {
			current.WriteByte(ch)
			if ch == '"' {
				inDoubleQuote = false
			}
			continue
		}

		switch ch {
		case '-':
			if i+1 < len(sqlStr) && sqlStr[i+1] == '-' {
				inLineComment = true
			}
			current.WriteByte(ch)
		case '/':
			if i+1 < len(sqlStr) && sqlStr[i+1] == '*' {
				inBlockComment = true
			}
			current.WriteByte(ch)
		case '\'':
			inSingleQuote = true
			current.WriteByte(ch)
		case '"':
			inDoubleQuote = true
			current.WriteByte(ch)
		case ';':
			s := strings.TrimSpace(current.String())
			if s != "" {
				statements = append(statements, s)
			}
			current.Reset()
		default:
			current.WriteByte(ch)
		}
	}

	s := strings.TrimSpace(current.String())
	if s != "" && !isOnlyComments(s) {
		statements = append(statements, s)
	}
	return statements
}

// isOnlyComments returns true if the string contains only SQL comments and whitespace.
func isOnlyComments(s string) bool {
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "--") || strings.HasPrefix(line, "/*") || strings.HasPrefix(line, "*/") || strings.HasPrefix(line, "*") {
			continue
		}
		return false
	}
	return true
}

// truncate shortens a string to maxLen characters, appending "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// PrintStatus logs a formatted table of migration statuses.
func PrintStatus(entries []StatusEntry) {
	fmt.Printf("%-8s %-50s %-12s %-8s %s\n", "VERSION", "NAME", "STATUS", "DIRTY", "APPLIED AT")
	fmt.Println(strings.Repeat("-", 110))
	for _, e := range entries {
		status := "pending"
		appliedAt := "-"
		dirty := "-"
		if e.Applied {
			status = "applied"
			dirty = fmt.Sprintf("%v", e.Dirty)
			if e.AppliedAt != nil {
				appliedAt = e.AppliedAt.Format(time.RFC3339)
			}
		}
		fmt.Printf("%-8d %-50s %-12s %-8s %s\n", e.Version, e.Name, status, dirty, appliedAt)
	}
}

// Keep the sql import used.
var _ *sql.DB
