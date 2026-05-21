-- 000008_add_job_taxonomy_tables.sql
-- Create departments and job_locations base-data tables,
-- extend jobs with foreign-key columns, seed initial data,
-- and backfill existing jobs where possible.
-- Does NOT drop or remove any existing columns.

-- ── 1. Departments table (tree-structured) ──────────────────────────────

CREATE TABLE IF NOT EXISTS `departments` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '部门ID',
  `parent_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '父部门ID，0表示根部门',
  `name` VARCHAR(64) NOT NULL COMMENT '部门名称',
  `full_name` VARCHAR(255) NOT NULL COMMENT '完整部门路径，如 技术研发部/后端组',
  `path` VARCHAR(512) NOT NULL COMMENT 'ID路径，如 /1/8/13/',
  `depth` INT NOT NULL DEFAULT 1 COMMENT '层级深度，根节点为1',
  `sort_order` INT NOT NULL DEFAULT 0 COMMENT '排序值，越小越靠前',
  `is_active` TINYINT NOT NULL DEFAULT 1 COMMENT '状态：1启用 0停用',
  `created_by` BIGINT UNSIGNED DEFAULT NULL COMMENT '创建管理员ID',
  `updated_by` BIGINT UNSIGNED DEFAULT NULL COMMENT '最后更新管理员ID',
  `deleted_at` DATETIME DEFAULT NULL COMMENT '逻辑删除时间',
  `deleted_by` BIGINT UNSIGNED DEFAULT NULL COMMENT '删除管理员ID',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_department_parent_name` (`parent_id`, `name`),
  KEY `idx_department_parent_sort` (`parent_id`, `sort_order`, `id`),
  KEY `idx_department_active` (`is_active`),
  KEY `idx_department_path` (`path`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='岗位部门基础数据表';

-- ── 2. Job locations table (flat list) ──────────────────────────────────

CREATE TABLE IF NOT EXISTS `job_locations` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '地点ID',
  `name` VARCHAR(128) NOT NULL COMMENT '地点名称',
  `code` VARCHAR(64) DEFAULT NULL COMMENT '地点编码，可选',
  `sort_order` INT NOT NULL DEFAULT 0 COMMENT '排序值，越小越靠前',
  `is_active` TINYINT NOT NULL DEFAULT 1 COMMENT '状态：1启用 0停用',
  `created_by` BIGINT UNSIGNED DEFAULT NULL COMMENT '创建管理员ID',
  `updated_by` BIGINT UNSIGNED DEFAULT NULL COMMENT '最后更新管理员ID',
  `deleted_at` DATETIME DEFAULT NULL COMMENT '逻辑删除时间',
  `deleted_by` BIGINT UNSIGNED DEFAULT NULL COMMENT '删除管理员ID',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_job_location_name` (`name`),
  KEY `idx_job_location_active_sort` (`is_active`, `sort_order`, `id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='岗位地点基础数据表';

-- ── 3. Extend jobs table with foreign-key columns ───────────────────────

ALTER TABLE `jobs`
  ADD COLUMN `department_id` BIGINT UNSIGNED DEFAULT NULL COMMENT '关联 departments.id' AFTER `department`,
  ADD COLUMN `location_id` BIGINT UNSIGNED DEFAULT NULL COMMENT '关联 job_locations.id' AFTER `location`,
  ADD KEY `idx_department_id` (`department_id`),
  ADD KEY `idx_location_id` (`location_id`);

-- ── 4. Seed initial department data ─────────────────────────────────────

INSERT INTO `departments` (`parent_id`, `name`, `full_name`, `path`, `depth`, `sort_order`, `is_active`) VALUES
  (0, '技术研发部', '技术研发部', '', 1, 1, 1),
  (0, '产品部',     '产品部',     '', 1, 2, 1),
  (0, '设计部',     '设计部',     '', 1, 3, 1),
  (0, '市场部',     '市场部',     '', 1, 4, 1),
  (0, '销售部',     '销售部',     '', 1, 5, 1),
  (0, '运营部',     '运营部',     '', 1, 6, 1),
  (0, '人力资源部', '人力资源部', '', 1, 7, 1),
  (0, '财务部',     '财务部',     '', 1, 8, 1),
  (0, '客户成功部', '客户成功部', '', 1, 9, 1);

-- After inserting root departments, update their `path` to include their own IDs.
UPDATE `departments` SET `path` = CONCAT('/', id, '/') WHERE parent_id = 0 AND path = '';

-- ── 5. Seed initial location data ────────────────────────────────────────

INSERT INTO `job_locations` (`name`, `code`, `sort_order`, `is_active`) VALUES
  ('北京', 'beijing',  1, 1),
  ('上海', 'shanghai', 2, 1),
  ('广州', 'guangzhou',3, 1),
  ('深圳', 'shenzhen', 4, 1),
  ('杭州', 'hangzhou', 5, 1),
  ('成都', 'chengdu',  6, 1),
  ('武汉', 'wuhan',    7, 1),
  ('西安', 'xian',     8, 1),
  ('南京', 'nanjing',  9, 1),
  ('远程', 'remote',  10, 1);

-- ── 6. Backfill existing jobs ────────────────────────────────────────────

-- Match by exact department name or full_name
UPDATE jobs j
JOIN departments d ON d.full_name = j.department OR d.name = j.department
SET j.department_id = d.id
WHERE j.department_id IS NULL
  AND j.department IS NOT NULL
  AND j.department <> '';

-- Match by exact location name
UPDATE jobs j
JOIN job_locations l ON l.name = j.location
SET j.location_id = l.id
WHERE j.location_id IS NULL
  AND j.location IS NOT NULL
  AND j.location <> '';
