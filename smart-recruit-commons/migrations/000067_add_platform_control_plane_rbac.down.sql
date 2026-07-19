DELETE mapping
FROM `role_permissions` mapping
JOIN `roles` role ON role.id = mapping.role_id
JOIN `permissions` permission ON permission.id = mapping.permission_id
WHERE role.role_key IN ('platform_admin', 'platform_operator', 'platform_auditor')
  AND permission.permission_key LIKE 'platform.%';

DELETE FROM `roles` WHERE `role_key` IN ('platform_operator', 'platform_auditor');
DELETE FROM `permissions` WHERE `permission_key` LIKE 'platform.%';
