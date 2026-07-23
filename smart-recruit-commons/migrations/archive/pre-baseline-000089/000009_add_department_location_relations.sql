-- 000009_add_department_location_relations.sql
-- Add department-location many-to-many association table,
-- inherit_locations flag on departments, initialise data
-- from historical jobs, and provide fallback coverage.

-- ── 1. Add inherit_locations column to departments ────────────────────

ALTER TABLE `departments`
  ADD COLUMN `inherit_locations` TINYINT NOT NULL DEFAULT 1 COMMENT '是否继承上级部门地点配置：1=继承 0=自定义' AFTER `is_active`;

-- ── 2. Create department_locations association table ──────────────────

CREATE TABLE IF NOT EXISTS `department_locations` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '部门地点关联ID',
  `department_id` BIGINT UNSIGNED NOT NULL COMMENT '部门ID',
  `location_id` BIGINT UNSIGNED NOT NULL COMMENT '地点ID',
  `is_active` TINYINT NOT NULL DEFAULT 1 COMMENT '状态：1启用 0停用',
  `created_by` BIGINT UNSIGNED DEFAULT NULL COMMENT '创建管理员ID',
  `updated_by` BIGINT UNSIGNED DEFAULT NULL COMMENT '最后更新管理员ID',
  `deleted_at` DATETIME DEFAULT NULL COMMENT '逻辑删除时间',
  `deleted_by` BIGINT UNSIGNED DEFAULT NULL COMMENT '删除管理员ID',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_department_location` (`department_id`, `location_id`),
  KEY `idx_department_active` (`department_id`, `is_active`, `deleted_at`),
  KEY `idx_location_active` (`location_id`, `is_active`, `deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='部门可用地点关联表';

-- ── 3. Backfill from historical jobs ──────────────────────────────────

INSERT INTO department_locations (`department_id`, `location_id`, `is_active`)
SELECT DISTINCT j.department_id, j.location_id, 1
FROM jobs j
WHERE j.department_id IS NOT NULL
  AND j.location_id IS NOT NULL
ON DUPLICATE KEY UPDATE
  is_active = 1,
  deleted_at = NULL,
  deleted_by = NULL;

-- ── 4. Fallback: give departments without location config deterministic sample locations ──

-- Root departments get a stable pseudo-random sample of 3 active locations.
-- This keeps initial data useful without assigning every location everywhere.
INSERT INTO department_locations (`department_id`, `location_id`, `is_active`)
WITH active_locations AS (
  SELECT
    id,
    ROW_NUMBER() OVER (ORDER BY sort_order, id) AS rn,
    COUNT(*) OVER () AS total_count
  FROM job_locations
  WHERE is_active = 1
    AND deleted_at IS NULL
),
root_targets AS (
  SELECT d.id AS department_id, l.id AS location_id
  FROM departments d
  JOIN active_locations l ON (
    l.rn = MOD(d.id - 1, l.total_count) + 1
    OR l.rn = MOD(d.id + 2, l.total_count) + 1
    OR l.rn = MOD(d.id + 5, l.total_count) + 1
  )
  WHERE d.deleted_at IS NULL
    AND d.parent_id = 0
    AND NOT EXISTS (
      SELECT 1
      FROM department_locations dl
      WHERE dl.department_id = d.id
        AND dl.deleted_at IS NULL
  )
)
SELECT department_id, location_id, 1
FROM root_targets
ON DUPLICATE KEY UPDATE
  is_active = 1,
  deleted_at = NULL,
  deleted_by = NULL;

-- Depth-2 departments inherit a subset of their parent department locations.
INSERT INTO department_locations (`department_id`, `location_id`, `is_active`)
WITH parent_locations AS (
  SELECT
    d.id AS department_id,
    dl.location_id,
    ROW_NUMBER() OVER (PARTITION BY d.id ORDER BY l.sort_order, l.id) AS rn
  FROM departments d
  JOIN department_locations dl ON dl.department_id = d.parent_id
    AND dl.is_active = 1
    AND dl.deleted_at IS NULL
  JOIN job_locations l ON l.id = dl.location_id
    AND l.is_active = 1
    AND l.deleted_at IS NULL
  WHERE d.deleted_at IS NULL
    AND d.parent_id <> 0
    AND d.depth = 2
    AND NOT EXISTS (
      SELECT 1
      FROM department_locations existing
      WHERE existing.department_id = d.id
        AND existing.deleted_at IS NULL
    )
)
SELECT department_id, location_id, 1
FROM parent_locations
WHERE rn <= 2
ON DUPLICATE KEY UPDATE
  is_active = 1,
  deleted_at = NULL,
  deleted_by = NULL;

-- Deeper departments use the same rule after depth-2 has been populated.
INSERT INTO department_locations (`department_id`, `location_id`, `is_active`)
WITH parent_locations AS (
  SELECT
    d.id AS department_id,
    dl.location_id,
    ROW_NUMBER() OVER (PARTITION BY d.id ORDER BY l.sort_order, l.id) AS rn
  FROM departments d
  JOIN department_locations dl ON dl.department_id = d.parent_id
    AND dl.is_active = 1
    AND dl.deleted_at IS NULL
  JOIN job_locations l ON l.id = dl.location_id
    AND l.is_active = 1
    AND l.deleted_at IS NULL
  WHERE d.deleted_at IS NULL
    AND d.parent_id <> 0
    AND d.depth > 2
    AND NOT EXISTS (
      SELECT 1
      FROM department_locations existing
      WHERE existing.department_id = d.id
        AND existing.deleted_at IS NULL
    )
)
SELECT department_id, location_id, 1
FROM parent_locations
WHERE rn <= 2
ON DUPLICATE KEY UPDATE
  is_active = 1,
  deleted_at = NULL,
  deleted_by = NULL;

-- ── 5. Initialise inherit_locations ────────────────────────────────────

-- Root departments (parent_id=0): set inherit_locations=0 (use own config).
UPDATE departments
SET inherit_locations = 0
WHERE deleted_at IS NULL
  AND parent_id = 0;

-- Non-root departments: set inherit_locations=1 (inherit from parent).
UPDATE departments
SET inherit_locations = 1
WHERE deleted_at IS NULL
  AND parent_id <> 0;
