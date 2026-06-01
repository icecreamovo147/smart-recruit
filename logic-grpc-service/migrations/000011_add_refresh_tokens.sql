-- Migration: 000011_add_refresh_tokens
-- Purpose: Opaque refresh token storage with rotation and reuse detection.
-- Access token remains a short-lived JWT; refresh token is a crypto/rand random string stored as sha256 hash.
-- Rotation: each successful refresh revokes the old token and issues a new one.
-- Reuse detection: if a revoked token is used again, the entire family is invalidated.

CREATE TABLE IF NOT EXISTS `refresh_tokens` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `user_id` bigint NOT NULL,
  `token_hash` char(64) NOT NULL COMMENT 'sha256 of the plaintext refresh token',
  `family_id` varchar(64) NOT NULL COMMENT 'login session group; rotation keeps family_id unchanged',
  `expires_at` datetime(3) NOT NULL COMMENT 'token expiry, matches RefreshTokenTTL (~30 days)',
  `revoked_at` datetime(3) NULL COMMENT 'set when this token is replaced by a new one',
  `replaced_by_hash` char(64) NULL COMMENT 'hash of the token that replaced this one',
  `reuse_detected_at` datetime(3) NULL COMMENT 'set when a revoked token is reused (potential attack)',
  `created_ip` varchar(64) NULL COMMENT 'original login IP for audit',
  `created_user_agent` varchar(255) NULL COMMENT 'original login User-Agent for audit',
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_refresh_tokens_token_hash` (`token_hash`),
  KEY `idx_refresh_tokens_user_id` (`user_id`),
  KEY `idx_refresh_tokens_family_id` (`family_id`),
  KEY `idx_refresh_tokens_expires_at` (`expires_at`),
  CONSTRAINT `fk_refresh_tokens_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`user_id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
