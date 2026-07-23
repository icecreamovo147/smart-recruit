-- Down: 000009_add_department_location_relations

DROP TABLE IF EXISTS `department_locations`;

ALTER TABLE `departments`
  DROP COLUMN `inherit_locations`;
