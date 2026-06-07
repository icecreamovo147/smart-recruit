-- Down: 000020_add_candidate_collaboration

DELETE rp FROM `role_permissions` rp
  INNER JOIN `permissions` p ON p.id = rp.permission_id
  WHERE p.permission_key IN (
    'collaboration.note.read', 'collaboration.note.create',
    'collaboration.tag.manage', 'collaboration.task.manage'
  );

DELETE FROM `permissions` WHERE `permission_key` IN (
  'collaboration.note.read', 'collaboration.note.create',
  'collaboration.tag.manage', 'collaboration.task.manage'
);

DROP TABLE IF EXISTS `follow_up_tasks`;
DROP TABLE IF EXISTS `candidate_tag_assignments`;
DROP TABLE IF EXISTS `candidate_tags`;
DROP TABLE IF EXISTS `candidate_notes`;
