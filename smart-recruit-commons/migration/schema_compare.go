package migration

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
)

type databaseSchemaDiff struct {
	Category string
	Name     string
	Field    string
	Left     string
	Right    string
}

// compareDatabaseSchemas compares schema structure while deliberately ignoring
// schema_migrations and all row data. It is used both by the baseline adoption
// gate and the MySQL consistency test.
func compareDatabaseSchemas(ctx context.Context, db *sql.DB, leftSchema, rightSchema string) (string, error) {
	var diffs []databaseSchemaDiff

	leftTables, err := queryDatabaseTableInfo(ctx, db, leftSchema)
	if err != nil {
		return "", err
	}
	rightTables, err := queryDatabaseTableInfo(ctx, db, rightSchema)
	if err != nil {
		return "", err
	}

	leftSet := tableInfoWithoutMigrationTable(leftTables)
	rightSet := tableInfoWithoutMigrationTable(rightTables)
	for table := range leftSet {
		rightTable, ok := rightSet[table]
		if !ok {
			diffs = append(diffs, databaseSchemaDiff{"tables", table, "exists", "yes", "no"})
			continue
		}
		if leftSet[table].Engine != rightTable.Engine {
			diffs = append(diffs, databaseSchemaDiff{"tables", table, "engine", leftSet[table].Engine, rightTable.Engine})
		}
		if leftSet[table].Collation != rightTable.Collation {
			diffs = append(diffs, databaseSchemaDiff{"tables", table, "collation", leftSet[table].Collation, rightTable.Collation})
		}
	}
	for table := range rightSet {
		if _, ok := leftSet[table]; !ok {
			diffs = append(diffs, databaseSchemaDiff{"tables", table, "exists", "no", "yes"})
		}
	}

	for table := range leftSet {
		if _, ok := rightSet[table]; !ok {
			continue
		}
		leftColumns, err := queryDatabaseColumnInfo(ctx, db, leftSchema, table)
		if err != nil {
			return "", err
		}
		rightColumns, err := queryDatabaseColumnInfo(ctx, db, rightSchema, table)
		if err != nil {
			return "", err
		}
		diffs = append(diffs, compareDatabaseColumns(table, leftColumns, rightColumns)...)

		leftIndexes, err := queryDatabaseIndexInfo(ctx, db, leftSchema, table)
		if err != nil {
			return "", err
		}
		rightIndexes, err := queryDatabaseIndexInfo(ctx, db, rightSchema, table)
		if err != nil {
			return "", err
		}
		diffs = append(diffs, compareDatabaseIndexes(table, leftIndexes, rightIndexes)...)
	}

	leftFKs, err := queryDatabaseForeignKeys(ctx, db, leftSchema)
	if err != nil {
		return "", err
	}
	rightFKs, err := queryDatabaseForeignKeys(ctx, db, rightSchema)
	if err != nil {
		return "", err
	}
	diffs = append(diffs, compareDefinitionMaps("foreign_keys", leftFKs, rightFKs)...)

	leftChecks, err := queryDatabaseChecks(ctx, db, leftSchema)
	if err != nil {
		return "", err
	}
	rightChecks, err := queryDatabaseChecks(ctx, db, rightSchema)
	if err != nil {
		return "", err
	}
	diffs = append(diffs, compareDefinitionMaps("checks", leftChecks, rightChecks)...)

	if len(diffs) == 0 {
		return "", nil
	}
	sort.Slice(diffs, func(i, j int) bool {
		if diffs[i].Category != diffs[j].Category {
			return diffs[i].Category < diffs[j].Category
		}
		if diffs[i].Name != diffs[j].Name {
			return diffs[i].Name < diffs[j].Name
		}
		return diffs[i].Field < diffs[j].Field
	})
	var b strings.Builder
	for _, diff := range diffs {
		fmt.Fprintf(&b, "  [%s] %s.%s: current=%q baseline=%q\n",
			diff.Category, diff.Name, diff.Field, diff.Left, diff.Right)
	}
	return b.String(), nil
}

type databaseTableInfo struct {
	Name      string
	Engine    string
	Collation string
}

func queryDatabaseTableInfo(ctx context.Context, db *sql.DB, schema string) ([]databaseTableInfo, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT LOWER(table_name), engine, table_collation
		 FROM information_schema.tables
		 WHERE table_schema = ? AND table_type = 'BASE TABLE'
		 ORDER BY table_name`, schema)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []databaseTableInfo
	for rows.Next() {
		var table databaseTableInfo
		if err := rows.Scan(&table.Name, &table.Engine, &table.Collation); err != nil {
			return nil, err
		}
		result = append(result, table)
	}
	return result, rows.Err()
}

func tableInfoWithoutMigrationTable(values []databaseTableInfo) map[string]databaseTableInfo {
	result := make(map[string]databaseTableInfo, len(values))
	for _, value := range values {
		if value.Name != "schema_migrations" {
			result[value.Name] = value
		}
	}
	return result
}

type databaseColumnInfo struct {
	Name       string
	Type       string
	Nullable   string
	Default    *string
	Extra      string
	Generation string
	Charset    sql.NullString
	Collation  sql.NullString
}

func queryDatabaseColumnInfo(ctx context.Context, db *sql.DB, schema, table string) ([]databaseColumnInfo, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT LOWER(column_name), column_type, is_nullable, column_default, extra,
		        generation_expression, character_set_name, collation_name
		 FROM information_schema.columns
		 WHERE table_schema = ? AND LOWER(table_name) = ?
		 ORDER BY ordinal_position`, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []databaseColumnInfo
	for rows.Next() {
		var column databaseColumnInfo
		var defaultValue sql.NullString
		if err := rows.Scan(
			&column.Name,
			&column.Type,
			&column.Nullable,
			&defaultValue,
			&column.Extra,
			&column.Generation,
			&column.Charset,
			&column.Collation,
		); err != nil {
			return nil, err
		}
		if defaultValue.Valid {
			column.Default = &defaultValue.String
		}
		result = append(result, column)
	}
	return result, rows.Err()
}

func compareDatabaseColumns(table string, left, right []databaseColumnInfo) []databaseSchemaDiff {
	leftMap := make(map[string]databaseColumnInfo, len(left))
	rightMap := make(map[string]databaseColumnInfo, len(right))
	for _, column := range left {
		leftMap[column.Name] = column
	}
	for _, column := range right {
		rightMap[column.Name] = column
	}

	var diffs []databaseSchemaDiff
	for name, leftColumn := range leftMap {
		rightColumn, ok := rightMap[name]
		fullName := table + "." + name
		if !ok {
			diffs = append(diffs, databaseSchemaDiff{"columns", fullName, "exists", "yes", "no"})
			continue
		}
		if leftColumn.Type != rightColumn.Type {
			diffs = append(diffs, databaseSchemaDiff{"columns", fullName, "type", leftColumn.Type, rightColumn.Type})
		}
		if leftColumn.Nullable != rightColumn.Nullable {
			diffs = append(diffs, databaseSchemaDiff{"columns", fullName, "nullable", leftColumn.Nullable, rightColumn.Nullable})
		}
		if !databaseDefaultsEqual(leftColumn.Default, rightColumn.Default) {
			diffs = append(diffs, databaseSchemaDiff{
				"columns", fullName, "default",
				databaseDefaultString(leftColumn.Default), databaseDefaultString(rightColumn.Default),
			})
		}
		if leftColumn.Extra != rightColumn.Extra {
			diffs = append(diffs, databaseSchemaDiff{"columns", fullName, "extra", leftColumn.Extra, rightColumn.Extra})
		}
		if leftColumn.Generation != rightColumn.Generation {
			diffs = append(diffs, databaseSchemaDiff{"columns", fullName, "generation", leftColumn.Generation, rightColumn.Generation})
		}
		if leftColumn.Charset != rightColumn.Charset {
			diffs = append(diffs, databaseSchemaDiff{"columns", fullName, "charset", leftColumn.Charset.String, rightColumn.Charset.String})
		}
		if leftColumn.Collation != rightColumn.Collation {
			diffs = append(diffs, databaseSchemaDiff{"columns", fullName, "collation", leftColumn.Collation.String, rightColumn.Collation.String})
		}
	}
	for name := range rightMap {
		if _, ok := leftMap[name]; !ok {
			diffs = append(diffs, databaseSchemaDiff{"columns", table + "." + name, "exists", "no", "yes"})
		}
	}
	return diffs
}

type databaseIndexInfo struct {
	Name       string
	Sequence   int
	Column     sql.NullString
	Expression sql.NullString
	NonUnique  bool
	IndexType  string
	SubPart    sql.NullInt64
}

func queryDatabaseIndexInfo(ctx context.Context, db *sql.DB, schema, table string) ([]databaseIndexInfo, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT LOWER(index_name), seq_in_index, LOWER(column_name), expression,
		        non_unique, index_type, sub_part
		 FROM information_schema.statistics
		 WHERE table_schema = ? AND LOWER(table_name) = ?
		 ORDER BY index_name, seq_in_index`, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []databaseIndexInfo
	for rows.Next() {
		var index databaseIndexInfo
		if err := rows.Scan(
			&index.Name,
			&index.Sequence,
			&index.Column,
			&index.Expression,
			&index.NonUnique,
			&index.IndexType,
			&index.SubPart,
		); err != nil {
			return nil, err
		}
		result = append(result, index)
	}
	return result, rows.Err()
}

func compareDatabaseIndexes(table string, left, right []databaseIndexInfo) []databaseSchemaDiff {
	leftMap := groupDatabaseIndexes(left)
	rightMap := groupDatabaseIndexes(right)
	var diffs []databaseSchemaDiff
	for name, leftDefinition := range leftMap {
		rightDefinition, ok := rightMap[name]
		fullName := table + "." + name
		if !ok {
			diffs = append(diffs, databaseSchemaDiff{"indexes", fullName, "exists", "yes", "no"})
			continue
		}
		leftString := databaseIndexDefinition(leftDefinition)
		rightString := databaseIndexDefinition(rightDefinition)
		if leftString != rightString {
			diffs = append(diffs, databaseSchemaDiff{"indexes", fullName, "definition", leftString, rightString})
		}
	}
	for name := range rightMap {
		if _, ok := leftMap[name]; !ok {
			diffs = append(diffs, databaseSchemaDiff{"indexes", table + "." + name, "exists", "no", "yes"})
		}
	}
	return diffs
}

func groupDatabaseIndexes(indexes []databaseIndexInfo) map[string][]databaseIndexInfo {
	result := make(map[string][]databaseIndexInfo)
	for _, index := range indexes {
		result[index.Name] = append(result[index.Name], index)
	}
	return result
}

func databaseIndexDefinition(indexes []databaseIndexInfo) string {
	var parts []string
	for _, index := range indexes {
		column := index.Column.String
		if !index.Column.Valid {
			column = "(" + index.Expression.String + ")"
		}
		if index.SubPart.Valid {
			column += fmt.Sprintf(":%d", index.SubPart.Int64)
		}
		parts = append(parts, fmt.Sprintf("%d:%s:%t:%s", index.Sequence, column, index.NonUnique, index.IndexType))
	}
	return strings.Join(parts, ",")
}

func queryDatabaseForeignKeys(ctx context.Context, db *sql.DB, schema string) (map[string]string, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT LOWER(kcu.table_name), LOWER(kcu.constraint_name), kcu.ordinal_position,
		        LOWER(kcu.column_name), LOWER(kcu.referenced_table_name),
		        LOWER(kcu.referenced_column_name), rc.update_rule, rc.delete_rule
		 FROM information_schema.key_column_usage kcu
		 JOIN information_schema.referential_constraints rc
		   ON rc.constraint_schema = kcu.constraint_schema
		  AND rc.table_name = kcu.table_name
		  AND rc.constraint_name = kcu.constraint_name
		 WHERE kcu.constraint_schema = ? AND kcu.referenced_table_name IS NOT NULL
		 ORDER BY kcu.table_name, kcu.constraint_name, kcu.ordinal_position`, schema)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]string)
	for rows.Next() {
		var table, constraint, column, referencedTable, referencedColumn, updateRule, deleteRule string
		var ordinal int
		if err := rows.Scan(
			&table, &constraint, &ordinal, &column, &referencedTable,
			&referencedColumn, &updateRule, &deleteRule,
		); err != nil {
			return nil, err
		}
		key := table + "." + constraint
		part := fmt.Sprintf("%d:%s>%s.%s:%s:%s",
			ordinal, column, referencedTable, referencedColumn, updateRule, deleteRule)
		if result[key] == "" {
			result[key] = part
		} else {
			result[key] += "," + part
		}
	}
	return result, rows.Err()
}

func queryDatabaseChecks(ctx context.Context, db *sql.DB, schema string) (map[string]string, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT LOWER(tc.table_name), LOWER(tc.constraint_name), cc.check_clause
		 FROM information_schema.table_constraints tc
		 JOIN information_schema.check_constraints cc
		   ON cc.constraint_schema = tc.constraint_schema
		  AND cc.constraint_name = tc.constraint_name
		 WHERE tc.constraint_schema = ? AND tc.constraint_type = 'CHECK'
		 ORDER BY tc.table_name, tc.constraint_name`, schema)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]string)
	for rows.Next() {
		var table, constraint, clause string
		if err := rows.Scan(&table, &constraint, &clause); err != nil {
			return nil, err
		}
		result[table+"."+constraint] = strings.Join(strings.Fields(strings.ToLower(clause)), " ")
	}
	return result, rows.Err()
}

func compareDefinitionMaps(category string, left, right map[string]string) []databaseSchemaDiff {
	var diffs []databaseSchemaDiff
	for name, leftDefinition := range left {
		rightDefinition, ok := right[name]
		if !ok {
			diffs = append(diffs, databaseSchemaDiff{category, name, "exists", "yes", "no"})
			continue
		}
		if leftDefinition != rightDefinition {
			diffs = append(diffs, databaseSchemaDiff{category, name, "definition", leftDefinition, rightDefinition})
		}
	}
	for name := range right {
		if _, ok := left[name]; !ok {
			diffs = append(diffs, databaseSchemaDiff{category, name, "exists", "no", "yes"})
		}
	}
	return diffs
}

func databaseDefaultsEqual(left, right *string) bool {
	if left == nil && right == nil {
		return true
	}
	if left == nil || right == nil {
		return false
	}
	return *left == *right
}

func databaseDefaultString(value *string) string {
	if value == nil {
		return "<nil>"
	}
	return *value
}

func queryDatabaseStrings(ctx context.Context, db *sql.DB, query string, args ...interface{}) ([]string, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []string
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, rows.Err()
}
