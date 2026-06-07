-- Down: 000007_add_invite_codes

ALTER TABLE `users`
  MODIFY COLUMN `role` TINYINT NOT NULL DEFAULT 1 COMMENT '角色：1=候选人 2=HR';

DROP TABLE IF EXISTS `invite_codes`;
