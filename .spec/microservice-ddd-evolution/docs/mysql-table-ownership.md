# MySQL Table Ownership

Smart Recruit keeps one physical MySQL instance. Service extraction uses logical table ownership, not database/schema splitting.

- Manifest: `smart-recruit-deploy/mysql-table-ownership.json`
- Check: `node scripts/check-mysql-table-ownership.mjs`

Every table in `db.sql` must have one owner with write access. Cross-service reads or writes must be declared in the manifest; new direct cross-service writes should be avoided in favor of RabbitMQ events and Outbox/Inbox flow.

The checker validates manifest coverage against `db.sql`, service names, owner write grants, and explicit GORM `Table(...)` usage in the extracted runtime roots. A service using a table without a declared grant fails the check, and a detected write requires an explicit `write` grant.
