-- Publishing a price may create orders and subscriptions immediately. Rolling it
-- back automatically would hide a purchased product and invalidate active commerce
-- references, so this data-publication migration is intentionally non-destructive.
SELECT 1;
