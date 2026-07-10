ALTER TABLE agent_skills
  ADD COLUMN agent_type VARCHAR(64) NOT NULL DEFAULT 'hr_recruiting_agent' AFTER trigger_keywords,
  ADD COLUMN category VARCHAR(64) NOT NULL DEFAULT 'general' AFTER agent_type,
  ADD COLUMN scenario VARCHAR(128) NOT NULL DEFAULT '' AFTER category,
  ADD COLUMN priority INT NOT NULL DEFAULT 0 AFTER scenario,
  ADD COLUMN risk_level VARCHAR(32) NOT NULL DEFAULT 'medium' AFTER priority,
  ADD COLUMN required_capabilities JSON NULL AFTER risk_level,
  ADD COLUMN output_schema JSON NULL AFTER required_capabilities,
  ADD COLUMN evaluation_criteria JSON NULL AFTER output_schema,
  ADD COLUMN semantic_tags JSON NULL AFTER evaluation_criteria,
  ADD KEY idx_agent_skills_governance (agent_type, category, priority);
