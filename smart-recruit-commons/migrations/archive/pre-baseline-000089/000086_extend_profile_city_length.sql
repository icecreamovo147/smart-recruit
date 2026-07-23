-- Extend city field to store province/city/district path.
ALTER TABLE `candidate_profiles`
  MODIFY COLUMN `city` VARCHAR(128) NULL COMMENT '所在/期望工作城市（省/市/区）';
