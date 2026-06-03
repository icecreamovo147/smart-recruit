-- 000013_seed_rbac_catalog.sql
-- Idempotently seeds system roles and permissions with UPSERT semantics.
-- This migration is safe to run multiple times.

-- ── Seed roles ─────────────────────────────────────────────────────────

INSERT INTO roles (role_key, name, description, is_system, created_at, updated_at) VALUES
('candidate',        '求职者',        '外部求职者，管理个人资料、简历、投递和AI会话', 1, NOW(), NOW()),
('recruiter',        '招聘专员',      '负责岗位发布、候选人流程、面试安排和HR AI使用', 1, NOW(), NOW()),
('recruiting_admin', '招聘管理员',    '管理招聘配置、邀请码、部门、地点、用户角色分配', 1, NOW(), NOW()),
('system_admin',     '系统管理员',    '管理平台安全配置、角色目录、权限目录、审计日志', 1, NOW(), NOW()),
('interviewer',      '面试官',        '查看被分配的面试并提交反馈', 1, NOW(), NOW())
ON DUPLICATE KEY UPDATE
  name = VALUES(name),
  description = VALUES(description),
  updated_at = NOW();

-- ── Seed permissions ───────────────────────────────────────────────────

INSERT INTO permissions (permission_key, resource, action, description, created_at, updated_at) VALUES
-- Auth
('auth.session.read',              'auth',      'read',   '查看自己的会话',             NOW(), NOW()),
-- Candidate
('candidate.profile.manage',       'candidate', 'manage', '管理候选人个人信息',         NOW(), NOW()),
('candidate.resume.manage',        'candidate', 'manage', '管理个人简历',               NOW(), NOW()),
('candidate.application.manage',   'candidate', 'manage', '创建和查看个人投递',         NOW(), NOW()),
-- Jobs
('job.read',                       'job',       'read',   '查看HR可见的岗位数据',       NOW(), NOW()),
('job.create',                     'job',       'create', '创建岗位',                   NOW(), NOW()),
('job.update',                     'job',       'update', '编辑岗位',                   NOW(), NOW()),
('job.publish',                    'job',       'publish','上下线岗位',                 NOW(), NOW()),
-- Applications
('application.read',               'application','read',  '查看范围内的候选人台账',     NOW(), NOW()),
('application.status.update',      'application','update','变更候选人状态',             NOW(), NOW()),
-- Interviews
('interview.read',                 'interview', 'read',   '查看分配的面试',             NOW(), NOW()),
('interview.schedule',             'interview', 'manage', '安排/修改/取消面试',         NOW(), NOW()),
('interview.feedback.submit',      'interview', 'create', '提交面试反馈',               NOW(), NOW()),
-- Notifications
('notification.read',              'notification','read', '查看自己的通知',             NOW(), NOW()),
-- AI
('ai.hr.use',                      'ai',        'use',    '使用HR AI助手',              NOW(), NOW()),
('ai.candidate.use',               'ai',        'use',    '使用候选人AI助手',           NOW(), NOW()),
-- Admin - recruiting
('admin.invite.manage',            'admin',     'manage', '管理邀请码',                 NOW(), NOW()),
('admin.department.manage',        'admin',     'manage', '管理部门及部门地点关联',     NOW(), NOW()),
('admin.location.manage',          'admin',     'manage', '管理工作地点',               NOW(), NOW()),
('admin.user.manage',              'admin',     'manage', '创建/修改/禁用员工账号',     NOW(), NOW()),
('admin.role.manage',              'admin',     'manage', '管理角色目录和权限分配',     NOW(), NOW()),
-- Audit
('audit.usage.read',               'audit',     'read',   '查看第三方/AI使用日志',      NOW(), NOW()),
('audit.security.read',            'audit',     'read',   '查看授权和安全审计事件',     NOW(), NOW()),
-- System
('system.config.manage',           'system',    'manage', '管理平台安全配置',            NOW(), NOW())
ON DUPLICATE KEY UPDATE
  resource = VALUES(resource),
  action = VALUES(action),
  description = VALUES(description),
  updated_at = NOW();

-- ── Seed role-permission mappings ──────────────────────────────────────
-- Uses sub-queries to look up role_id and permission_id by keys so this is
-- safe to re-run after any prior seed.

-- Candidate permissions
INSERT IGNORE INTO role_permissions (role_id, permission_id, created_at)
SELECT r.id, p.id, NOW()
FROM roles r, permissions p
WHERE r.role_key = 'candidate'
  AND p.permission_key IN (
    'auth.session.read',
    'candidate.profile.manage',
    'candidate.resume.manage',
    'candidate.application.manage',
    'notification.read',
    'ai.candidate.use'
  );

-- Recruiter permissions
INSERT IGNORE INTO role_permissions (role_id, permission_id, created_at)
SELECT r.id, p.id, NOW()
FROM roles r, permissions p
WHERE r.role_key = 'recruiter'
  AND p.permission_key IN (
    'auth.session.read',
    'job.read', 'job.create', 'job.update', 'job.publish',
    'application.read', 'application.status.update',
    'interview.read', 'interview.schedule', 'interview.feedback.submit',
    'notification.read',
    'ai.hr.use'
  );

-- Recruiting Admin permissions
INSERT IGNORE INTO role_permissions (role_id, permission_id, created_at)
SELECT r.id, p.id, NOW()
FROM roles r, permissions p
WHERE r.role_key = 'recruiting_admin'
  AND p.permission_key IN (
    'auth.session.read',
    'job.read', 'application.read',
    'notification.read',
    'ai.hr.use',
    'admin.invite.manage',
    'admin.department.manage',
    'admin.location.manage',
    'admin.user.manage',
    'admin.role.manage',
    'audit.usage.read'
  );

-- System Admin permissions (full platform)
INSERT IGNORE INTO role_permissions (role_id, permission_id, created_at)
SELECT r.id, p.id, NOW()
FROM roles r, permissions p
WHERE r.role_key = 'system_admin'
  AND p.permission_key IN (
    'auth.session.read',
    'admin.user.manage',
    'admin.role.manage',
    'audit.usage.read',
    'audit.security.read',
    'system.config.manage'
  );

-- Interviewer permissions
INSERT IGNORE INTO role_permissions (role_id, permission_id, created_at)
SELECT r.id, p.id, NOW()
FROM roles r, permissions p
WHERE r.role_key = 'interviewer'
  AND p.permission_key IN (
    'auth.session.read',
    'interview.read',
    'interview.feedback.submit',
    'notification.read'
  );
