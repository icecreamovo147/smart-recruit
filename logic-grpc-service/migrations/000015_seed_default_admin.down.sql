-- Down: 000015_seed_default_admin
-- Remove admin user's RBAC assignments and the admin user itself.

DELETE FROM `user_data_scopes` WHERE `user_id` = (SELECT `id` FROM `users` WHERE `username` = 'admin' LIMIT 1);
DELETE FROM `user_roles` WHERE `user_id` = (SELECT `id` FROM `users` WHERE `username` = 'admin' LIMIT 1);
DELETE FROM `users` WHERE `username` = 'admin';
