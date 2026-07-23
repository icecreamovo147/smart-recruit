ALTER TABLE `candidate_match_evaluations`
  DROP KEY `idx_candidate_match_capability_version`,
  DROP COLUMN `capability_snapshot_hash`,
  DROP COLUMN `capability_version_id`,
  DROP COLUMN `model_fallback_reason`,
  DROP COLUMN `effective_model_id`,
  DROP COLUMN `requested_model_id`;

ALTER TABLE `resume_parse_runs`
  DROP KEY `idx_resume_parse_capability_version`,
  DROP COLUMN `capability_snapshot_hash`,
  DROP COLUMN `capability_version_id`,
  DROP COLUMN `model_fallback_reason`,
  DROP COLUMN `effective_model_id`,
  DROP COLUMN `requested_model_id`;
