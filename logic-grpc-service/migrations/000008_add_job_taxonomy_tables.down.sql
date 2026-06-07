-- Down: 000008_add_job_taxonomy_tables

ALTER TABLE `jobs`
  DROP KEY `idx_location_id`,
  DROP KEY `idx_department_id`,
  DROP COLUMN `location_id`,
  DROP COLUMN `department_id`;

DROP TABLE IF EXISTS `job_locations`;
DROP TABLE IF EXISTS `departments`;
