-- 000012_add_rbac_schema.sql
-- Adds RBAC tables and new columns to users for the role-permission redesign.
-- All new tables use IF NOT EXISTS for idempotent execution.

-- ── Users table changes ───────────────────────────────────────────────

ALTER TABLE users
  ADD COLUMN account_type VARCHAR(32) NOT NULL DEFAULT 'candidate',
  ADD COLUMN status VARCHAR(32) NOT NULL DEFAULT 'active',
  ADD COLUMN token_version INT NOT NULL DEFAULT 1;

-- ── Roles ─────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS roles (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  role_key VARCHAR(64) NOT NULL,
  name VARCHAR(128) NOT NULL,
  description VARCHAR(512) NULL,
  is_system TINYINT NOT NULL DEFAULT 1,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_roles_role_key (role_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ── Permissions ───────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS permissions (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  permission_key VARCHAR(128) NOT NULL,
  resource VARCHAR(64) NOT NULL,
  action VARCHAR(64) NOT NULL,
  description VARCHAR(512) NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_permissions_permission_key (permission_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ── Role-Permission mappings ──────────────────────────────────────────

CREATE TABLE IF NOT EXISTS role_permissions (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  role_id BIGINT UNSIGNED NOT NULL,
  permission_id BIGINT UNSIGNED NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_role_permission (role_id, permission_id),
  KEY idx_permission_id (permission_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ── User-Role assignments ─────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS user_roles (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  user_id BIGINT UNSIGNED NOT NULL,
  role_id BIGINT UNSIGNED NOT NULL,
  assigned_by BIGINT UNSIGNED NULL,
  assigned_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  revoked_at DATETIME NULL,
  active_key TINYINT GENERATED ALWAYS AS (CASE WHEN revoked_at IS NULL THEN 1 ELSE NULL END) STORED,
  PRIMARY KEY (id),
  UNIQUE KEY uk_user_role_active (user_id, role_id, active_key),
  KEY idx_user_roles_user (user_id),
  KEY idx_user_roles_role (role_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ── User data scopes ──────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS user_data_scopes (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  user_id BIGINT UNSIGNED NOT NULL,
  scope_key VARCHAR(64) NOT NULL,
  resource_type VARCHAR(64) NOT NULL DEFAULT '',
  resource_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
  assigned_by BIGINT UNSIGNED NULL,
  assigned_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  revoked_at DATETIME NULL,
  active_key TINYINT GENERATED ALWAYS AS (CASE WHEN revoked_at IS NULL THEN 1 ELSE NULL END) STORED,
  PRIMARY KEY (id),
  UNIQUE KEY uk_user_scope_active (user_id, scope_key, resource_type, resource_id, active_key),
  KEY idx_user_scope (user_id, scope_key, revoked_at),
  KEY idx_scope_resource (scope_key, resource_type, resource_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ── Authorization audit logs ──────────────────────────────────────────

CREATE TABLE IF NOT EXISTS authorization_audit_logs (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  actor_user_id BIGINT UNSIGNED NOT NULL,
  actor_roles VARCHAR(512) NOT NULL,
  permission_key VARCHAR(128) NOT NULL,
  resource_type VARCHAR(64) NOT NULL,
  resource_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
  decision VARCHAR(16) NOT NULL,
  reason VARCHAR(512) NULL,
  request_id VARCHAR(64) NULL,
  client_ip VARCHAR(64) NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_actor_created (actor_user_id, created_at),
  KEY idx_permission_created (permission_key, created_at),
  KEY idx_decision_created (decision, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
