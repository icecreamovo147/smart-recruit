-- The normalized records were originally declared immediately effective.
-- Reintroducing timezone skew on rollback would recreate the access outage, so
-- this data correction is intentionally retained.
SELECT 1;
