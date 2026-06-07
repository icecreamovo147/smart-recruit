-- Down: 000019_add_offer_tables

DELETE rp FROM `role_permissions` rp
  INNER JOIN `permissions` p ON p.id = rp.permission_id
  WHERE p.permission_key IN ('offer.read', 'offer.manage', 'offer.send', 'offer.decision.manage');

DELETE FROM `permissions` WHERE `permission_key` IN (
  'offer.read', 'offer.manage', 'offer.send', 'offer.decision.manage'
);

DROP TABLE IF EXISTS `offer_events`;
DROP TABLE IF EXISTS `offers`;
