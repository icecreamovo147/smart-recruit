# Database migration baseline

Version `000089` is the immutable schema baseline. Migrations `000001` through
`000088` remain available under `archive/pre-baseline-000089/` for checksum
verification, audit, and older-branch compatibility, but the active runner does
not execute them on a fresh database.

## Supported database paths

- Fresh database: run the migration command normally. The runner applies
  `000089_schema_baseline.sql`, followed by any future migration.
- Database imported from the matching `db.sql` with no migration history: run
  `go run ./cmd/migrate --baseline 89`.
- Existing database with complete versions `1` through `88`: back it up, then
  run `go run ./cmd/migrate --adopt-baseline 89`.

Baseline adoption does not replay schema SQL against the live database. It:

1. Requires a clean, contiguous `1..88` history and validates every archived
   checksum, including explicitly registered legacy checksums.
2. Executes the frozen baseline in a generated temporary database.
3. Compares tables, columns, generated expressions, indexes, foreign keys, and
   check constraints against the live database.
4. Inserts only the version `89` row into `schema_migrations`.

The adopting MySQL account therefore needs temporary `CREATE DATABASE` and
`DROP DATABASE` privileges. A schema difference fails before the version `89`
record is written. Repair such drift with a reviewed, data-preserving DDL
operation; do not edit historical SQL or manually forge migration history.

## Adding schema changes

Start new migrations at `000090`. Add an up/down pair, update `db.sql`, and run:

```bash
node scripts/check-migration-baseline.mjs
cd smart-recruit-commons
go test ./migration/...
MYSQL_DSN='...' go test -tags=mysql \
  -run 'TestMySQLMigrationConsistency|TestBaselineScenarios' ./migration/
```

Never edit `000089_schema_baseline.sql`, `baseline-lock.json`, or the archived
files as part of an ordinary schema change. The baseline intentionally has no
down migration because rolling it back would destroy the whole schema.
