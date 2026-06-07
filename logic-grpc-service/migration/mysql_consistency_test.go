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

	_ "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// TestMySQLMigrationConsistency verifies that applying all migrations produces
// the same schema as importing db.sql.
//
// Run with: go test -tags=mysql -run TestMySQLMigrationConsistency
// Requires MYSQL_DSN env var pointing to a MySQL 8.0 instance.
// Two temporary databases are created and destroyed by the test.
func TestMySQLMigrationConsistency(t *testing.T) {
	baseDSN := os.Getenv("MYSQL_DSN")
	if baseDSN == "" {
		t.Skip("MYSQL_DSN not set; skipping MySQL consistency test")
	}

	ctx := context.Background()

	// ── Connect ──────────────────────────────────────────────────────
	rootDB, err := gorm.Open(mysql.Open(baseDSN), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect mysql: %v", err)
	}
	rootSQL, _ := rootDB.DB()
	defer rootSQL.Close()

	dbNameMig := "recruitment_mig_test"
	dbNameSQL := "recruitment_dbsql_test"

	// Clean up any leftovers from a previous failed run.
	rootSQL.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", dbNameMig))
	rootSQL.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", dbNameSQL))

	if _, err := rootSQL.Exec(fmt.Sprintf("CREATE DATABASE `%s`", dbNameMig)); err != nil {
		t.Fatalf("create db %s: %v", dbNameMig, err)
	}
	if _, err := rootSQL.Exec(fmt.Sprintf("CREATE DATABASE `%s`", dbNameSQL)); err != nil {
		t.Fatalf("create db %s: %v", dbNameSQL, err)
	}

	defer func() {
		rootSQL.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", dbNameMig))
		rootSQL.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", dbNameSQL))
	}()

	// Build DSNs for each database by swapping the database name.
	dsnMig := replaceDBName(baseDSN, dbNameMig)
	dsnSQL := replaceDBName(baseDSN, dbNameSQL)

	// ── Database A: apply all migrations ─────────────────────────────
	dbMig, err := gorm.Open(mysql.Open(dsnMig), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("connect mig db: %v", err)
	}
	migSQL, _ := dbMig.DB()
	defer migSQL.Close()

	runner, err := NewRunner(dbMig, testMigrationsFS, ".")
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	if err := runner.Up(ctx); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	// ── Database B: import db.sql ────────────────────────────────────
	// Resolve db.sql relative to the project root (two levels up from
	// logic-grpc-service/migration/).
	dbSQLPath := filepath.Join("..", "..", "db.sql")
	dbSQLBytes, err := os.ReadFile(dbSQLPath)
	if err != nil {
		t.Fatalf("read db.sql: %v", err)
	}

	// Execute db.sql against the test database.
	// Split on semicolons and execute each statement.
	statements := splitStatements(string(dbSQLBytes))
	dbSQLConn, err := gorm.Open(mysql.Open(dsnSQL), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("connect dbsql db: %v", err)
	}
	sqlDBConn, _ := dbSQLConn.DB()
	defer sqlDBConn.Close()

	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		// Skip USE and CREATE DATABASE — we connect directly to the test database.
		if isIgnoredDBStatement(stmt) {
			continue
		}
		if _, err := sqlDBConn.ExecContext(ctx, stmt); err != nil {
			t.Fatalf("exec db.sql stmt %q: %v", truncate(stmt, 80), err)
		}
	}

	// ── Compare information_schema ───────────────────────────────────
	diff, err := compareSchemas(ctx, rootSQL, dbNameMig, dbNameSQL)
	if err != nil {
		t.Fatalf("compare schemas: %v", err)
	}
	if diff != "" {
		t.Errorf("Schema mismatch between migrations and db.sql:\n%s", diff)
	}
}

// replaceDBName replaces the database name in a MySQL DSN.
// DSN format: user:pass@tcp(host:port)/dbname?params
func replaceDBName(dsn, newDB string) string {
	// Find the "/" after @tcp(...) or @unix(...)
	slashIdx := strings.Index(dsn, "/")
	if slashIdx < 0 {
		return dsn
	}
	rest := dsn[slashIdx+1:]
	// Find the first ? or / or end
	qIdx := strings.IndexAny(rest, "?/")
	if qIdx < 0 {
		return dsn[:slashIdx+1] + newDB
	}
	return dsn[:slashIdx+1] + newDB + rest[qIdx:]
}

// schemaDiff captures a single schema difference.
type schemaDiff struct {
	Category string // tables / columns / indexes
	Name     string
	Field    string
	MigVal   string
	SQLVal   string
}

// compareSchemas compares information_schema across two databases.
func compareSchemas(ctx context.Context, db *sql.DB, dbMig, dbSQL string) (string, error) {
	var diffs []schemaDiff

	// ── Compare tables ───────────────────────────────────────────────
	tablesMig, err := queryStrings(ctx, db,
		"SELECT LOWER(table_name) FROM information_schema.tables WHERE table_schema = ? AND table_type = 'BASE TABLE' ORDER BY table_name",
		dbMig)
	if err != nil {
		return "", err
	}
	tablesSQL, err := queryStrings(ctx, db,
		"SELECT LOWER(table_name) FROM information_schema.tables WHERE table_schema = ? AND table_type = 'BASE TABLE' ORDER BY table_name",
		dbSQL)
	if err != nil {
		return "", err
	}

	migSet := make(map[string]bool)
	for _, t := range tablesMig {
		if t == "schema_migrations" {
			continue
		}
		migSet[t] = true
	}
	sqlSet := make(map[string]bool)
	for _, t := range tablesSQL {
		if t == "schema_migrations" {
			continue
		}
		sqlSet[t] = true
	}

	// Tables in mig but not in sql
	for t := range migSet {
		if !sqlSet[t] {
			diffs = append(diffs, schemaDiff{Category: "tables", Name: t, Field: "exists", MigVal: "yes", SQLVal: "no"})
		}
	}
	for t := range sqlSet {
		if !migSet[t] {
			diffs = append(diffs, schemaDiff{Category: "tables", Name: t, Field: "exists", MigVal: "no", SQLVal: "yes"})
		}
	}

	// ── Compare columns for each table ───────────────────────────────
	for t := range migSet {
		if !sqlSet[t] {
			continue
		}
		colsMig, err := queryColumnInfo(ctx, db, dbMig, t)
		if err != nil {
			return "", err
		}
		colsSQL, err := queryColumnInfo(ctx, db, dbSQL, t)
		if err != nil {
			return "", err
		}
		diffs = append(diffs, compareColumnSets(t, colsMig, colsSQL)...)
	}

	// ── Compare indexes for each table ───────────────────────────────
	for t := range migSet {
		if !sqlSet[t] {
			continue
		}
		idxsMig, err := queryIndexInfo(ctx, db, dbMig, t)
		if err != nil {
			return "", err
		}
		idxsSQL, err := queryIndexInfo(ctx, db, dbSQL, t)
		if err != nil {
			return "", err
		}
		diffs = append(diffs, compareIndexSets(t, idxsMig, idxsSQL)...)
	}

	if len(diffs) == 0 {
		return "", nil
	}

	var b strings.Builder
	for _, d := range diffs {
		fmt.Fprintf(&b, "  [%s] %s.%s: mig=%q sql=%q\n", d.Category, d.Name, d.Field, d.MigVal, d.SQLVal)
	}
	return b.String(), nil
}

type columnInfo struct {
	Name     string
	Type     string
	Nullable string
	Default  *string
	Extra    string
}

func queryColumnInfo(ctx context.Context, db *sql.DB, schema, table string) ([]columnInfo, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT LOWER(column_name), column_type, is_nullable, column_default, extra
		 FROM information_schema.columns
		 WHERE table_schema = ? AND LOWER(table_name) = ?
		 ORDER BY ordinal_position`, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cols []columnInfo
	for rows.Next() {
		var c columnInfo
		var def sql.NullString
		if err := rows.Scan(&c.Name, &c.Type, &c.Nullable, &def, &c.Extra); err != nil {
			return nil, err
		}
		if def.Valid {
			c.Default = &def.String
		}
		cols = append(cols, c)
	}
	return cols, rows.Err()
}

type indexInfo struct {
	KeyName    string
	SeqInIndex int
	ColumnName string
	NonUnique  bool
	IndexType  string
}

func queryIndexInfo(ctx context.Context, db *sql.DB, schema, table string) ([]indexInfo, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT LOWER(index_name), seq_in_index, LOWER(column_name), non_unique, index_type
		 FROM information_schema.statistics
		 WHERE table_schema = ? AND LOWER(table_name) = ?
		 ORDER BY index_name, seq_in_index`, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var idxs []indexInfo
	for rows.Next() {
		var idx indexInfo
		if err := rows.Scan(&idx.KeyName, &idx.SeqInIndex, &idx.ColumnName, &idx.NonUnique, &idx.IndexType); err != nil {
			return nil, err
		}
		idxs = append(idxs, idx)
	}
	return idxs, rows.Err()
}

func compareColumnSets(table string, mig, sql []columnInfo) []schemaDiff {
	var diffs []schemaDiff
	migMap := make(map[string]columnInfo)
	for _, c := range mig {
		migMap[c.Name] = c
	}
	sqlMap := make(map[string]columnInfo)
	for _, c := range sql {
		sqlMap[c.Name] = c
	}

	for name, mc := range migMap {
		sc, ok := sqlMap[name]
		if !ok {
			diffs = append(diffs, schemaDiff{Category: "columns", Name: table + "." + name, Field: "exists", MigVal: "yes", SQLVal: "no"})
			continue
		}
		if mc.Type != sc.Type {
			diffs = append(diffs, schemaDiff{Category: "columns", Name: table + "." + name, Field: "type", MigVal: mc.Type, SQLVal: sc.Type})
		}
		if mc.Nullable != sc.Nullable {
			diffs = append(diffs, schemaDiff{Category: "columns", Name: table + "." + name, Field: "nullable", MigVal: mc.Nullable, SQLVal: sc.Nullable})
		}
		if !defaultsEqual(mc.Default, sc.Default) {
			dv1, dv2 := "<nil>", "<nil>"
			if mc.Default != nil {
				dv1 = *mc.Default
			}
			if sc.Default != nil {
				dv2 = *sc.Default
			}
			diffs = append(diffs, schemaDiff{Category: "columns", Name: table + "." + name, Field: "default", MigVal: dv1, SQLVal: dv2})
		}
		if mc.Extra != sc.Extra {
			diffs = append(diffs, schemaDiff{Category: "columns", Name: table + "." + name, Field: "extra", MigVal: mc.Extra, SQLVal: sc.Extra})
		}
	}
	for name := range sqlMap {
		if _, ok := migMap[name]; !ok {
			diffs = append(diffs, schemaDiff{Category: "columns", Name: table + "." + name, Field: "exists", MigVal: "no", SQLVal: "yes"})
		}
	}
	return diffs
}

func compareIndexSets(table string, mig, sql []indexInfo) []schemaDiff {
	// Group by key name
	migMap := groupIndexes(mig)
	sqlMap := groupIndexes(sql)

	var diffs []schemaDiff
	for name, mcols := range migMap {
		scols, ok := sqlMap[name]
		if !ok {
			diffs = append(diffs, schemaDiff{Category: "indexes", Name: table + "." + name, Field: "exists", MigVal: "yes", SQLVal: "no"})
			continue
		}
		if !indexColsEqual(mcols, scols) {
			diffs = append(diffs, schemaDiff{Category: "indexes", Name: table + "." + name, Field: "definition", MigVal: indexColsStr(mcols), SQLVal: indexColsStr(scols)})
		}
	}
	for name := range sqlMap {
		if _, ok := migMap[name]; !ok {
			diffs = append(diffs, schemaDiff{Category: "indexes", Name: table + "." + name, Field: "exists", MigVal: "no", SQLVal: "yes"})
		}
	}
	return diffs
}

func groupIndexes(idxs []indexInfo) map[string][]indexInfo {
	m := make(map[string][]indexInfo)
	for _, idx := range idxs {
		if idx.KeyName == "primary" {
			continue
		}
		m[idx.KeyName] = append(m[idx.KeyName], idx)
	}
	return m
}

func indexColsEqual(a, b []indexInfo) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].ColumnName != b[i].ColumnName || a[i].NonUnique != b[i].NonUnique || a[i].SeqInIndex != b[i].SeqInIndex {
			return false
		}
	}
	return true
}

func indexColsStr(idxs []indexInfo) string {
	var parts []string
	for _, idx := range idxs {
		u := "UNIQUE"
		if idx.NonUnique {
			u = "KEY"
		}
		parts = append(parts, fmt.Sprintf("%s(%s)", u, idx.ColumnName))
	}
	return strings.Join(parts, ",")
}

func defaultsEqual(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func queryStrings(ctx context.Context, db *sql.DB, query string, args ...interface{}) ([]string, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

// isIgnoredDBStatement returns true for USE and CREATE DATABASE statements
// that should be skipped when importing db.sql into a pre-connected database.
func isIgnoredDBStatement(stmt string) bool {
	s := strings.TrimSpace(stmt)
	// Normalise to uppercase for prefix check.
	upper := ""
	for _, ch := range s {
		if ch >= 'a' && ch <= 'z' {
			upper += string(ch - 32)
		} else if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' || ch == '`' {
			upper += " "
		} else {
			upper += string(ch)
		}
	}
	// Re-collapse spaces
	fields := strings.Fields(upper)
	if len(fields) >= 2 && fields[0] == "USE" {
		return true
	}
	if len(fields) >= 3 && fields[0] == "CREATE" && fields[1] == "DATABASE" {
		return true
	}
	return false
}
