-- Down: 000013_seed_rbac_catalog
-- Removes seeded role-permission mappings, permissions, and roles.
-- Only removes system (is_system=1) entries seeded by this migration.

DELETE rp FROM `role_permissions` rp
  INNER JOIN `roles` r ON r.id = rp.role_id
  WHERE r.is_system = 1;

DELETE FROM `permissions` WHERE `permission_key` IN (
  'auth.session.read',
  'candidate.profile.manage', 'candidate.resume.manage', 'candidate.application.manage',
  'job.read', 'job.create', 'job.update', 'job.publish',
  'application.read', 'application.status.update',
  'interview.read', 'interview.schedule', 'interview.feedback.submit',
  'notification.read',
  'ai.hr.use', 'ai.candidate.use',
  'admin.invite.manage', 'admin.department.manage', 'admin.location.manage',
  'admin.user.manage', 'admin.role.manage',
  'audit.usage.read', 'audit.security.read',
  'system.config.manage'
);

DELETE FROM `roles` WHERE `is_system` = 1;
