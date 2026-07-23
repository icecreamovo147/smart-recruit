-- P0: candidate profile screening fields
ALTER TABLE `candidate_profiles`
  ADD COLUMN `city` VARCHAR(64) NULL COMMENT '所在/期望工作城市' AFTER `skills`,
  ADD COLUMN `years_of_experience` DECIMAL(4,1) NULL COMMENT '工作年限' AFTER `city`,
  ADD COLUMN `job_status` VARCHAR(32) NULL COMMENT '求职状态: employed/resigned/fresh_graduate/student' AFTER `years_of_experience`,
  ADD COLUMN `expected_position` VARCHAR(128) NULL COMMENT '期望岗位' AFTER `job_status`,
  ADD COLUMN `expected_salary_min` INT NULL COMMENT '期望月薪下限（元）' AFTER `expected_position`,
  ADD COLUMN `expected_salary_max` INT NULL COMMENT '期望月薪上限（元）' AFTER `expected_salary_min`,
  ADD COLUMN `available_from` DATE NULL COMMENT '可到岗日期' AFTER `expected_salary_max`,
  ADD COLUMN `summary` VARCHAR(500) NULL COMMENT '个人简介' AFTER `available_from`;
