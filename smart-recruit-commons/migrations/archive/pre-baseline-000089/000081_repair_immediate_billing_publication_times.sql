-- Runtime admin saves previously supplied Go time.Time values through a
-- loc=Local MySQL connection while Billing compares DATETIME values against
-- UTC_TIMESTAMP(). Repair only immediate publications whose effective time
-- matches their creation time; intentionally scheduled versions are excluded.

UPDATE `billing_price_versions`
SET `effective_at` = UTC_TIMESTAMP(3),
    `updated_at` = UTC_TIMESTAMP(3)
WHERE `status` = 'published'
  AND `effective_at` > UTC_TIMESTAMP(3)
  AND ABS(TIMESTAMPDIFF(SECOND, `created_at`, `effective_at`)) <= 5;

UPDATE `ai_rate_cards`
SET `effective_at` = UTC_TIMESTAMP(3)
WHERE `status` = 'published'
  AND `effective_at` > UTC_TIMESTAMP(3)
  AND ABS(TIMESTAMPDIFF(SECOND, `created_at`, `effective_at`)) <= 5;
