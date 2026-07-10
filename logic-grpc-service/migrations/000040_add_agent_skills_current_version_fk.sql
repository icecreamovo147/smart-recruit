-- Clean orphaned current_version_id references before adding FK.
UPDATE agent_skills s
LEFT JOIN agent_skill_versions v ON v.id = s.current_version_id
SET s.current_version_id = NULL
WHERE s.current_version_id IS NOT NULL
  AND v.id IS NULL;

ALTER TABLE agent_skills
  ADD CONSTRAINT fk_agent_skills_current_version
  FOREIGN KEY (current_version_id)
  REFERENCES agent_skill_versions(id)
  ON DELETE SET NULL;
