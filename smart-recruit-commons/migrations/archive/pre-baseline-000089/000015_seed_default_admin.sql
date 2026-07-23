-- Idempotently seed the default admin user with RBAC assignments.
-- This mirrors the seed in db.sql to keep migrations self-contained for
-- developer workflows. For production fresh-start, use db.sql directly.

INSERT INTO `users` (`username`, `password`, `role`, `account_type`, `status`, `token_version`)
VALUES ('admin', '$2b$10$qxetp5jT6U7U5dd/k1G/v.qJ.FDlqFLWO3LHKv8Kwt6c49VXhLhOy', 3, 'staff', 'active', 1)
ON DUPLICATE KEY UPDATE `account_type` = 'staff';

-- Assign recruiting_admin role
INSERT IGNORE INTO `user_roles` (`user_id`, `role_id`, `assigned_at`)
SELECT u.id, r.id, NOW()
FROM `users` u, `roles` r
WHERE u.username = 'admin' AND r.role_key = 'recruiting_admin';

-- Assign recruiter role
INSERT IGNORE INTO `user_roles` (`user_id`, `role_id`, `assigned_at`)
SELECT u.id, r.id, NOW()
FROM `users` u, `roles` r
WHERE u.username = 'admin' AND r.role_key = 'recruiter';

-- Assign recruiting_all data scope
INSERT IGNORE INTO `user_data_scopes` (`user_id`, `scope_key`, `resource_type`, `resource_id`, `assigned_at`)
SELECT u.id, 'recruiting_all', '', 0, NOW()
FROM `users` u
WHERE u.username = 'admin';

-- Assign system_admin role (system management: audit log access, system config, etc.)
INSERT IGNORE INTO `user_roles` (`user_id`, `role_id`, `assigned_at`)
SELECT u.id, r.id, NOW()
FROM `users` u, `roles` r
WHERE u.username = 'admin' AND r.role_key = 'system_admin';
