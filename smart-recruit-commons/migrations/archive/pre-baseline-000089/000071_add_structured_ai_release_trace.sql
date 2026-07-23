ALTER TABLE `resume_parse_runs`
  ADD COLUMN `requested_model_id` BIGINT NULL AFTER `agent_run_id`,
  ADD COLUMN `effective_model_id` BIGINT NULL AFTER `requested_model_id`,
  ADD COLUMN `model_fallback_reason` VARCHAR(64) NULL AFTER `effective_model_id`,
  ADD COLUMN `capability_version_id` BIGINT NULL AFTER `model_fallback_reason`,
  ADD COLUMN `capability_snapshot_hash` CHAR(64) NULL AFTER `capability_version_id`,
  ADD KEY `idx_resume_parse_capability_version` (`capability_version_id`);

ALTER TABLE `candidate_match_evaluations`
  ADD COLUMN `requested_model_id` BIGINT NULL AFTER `agent_run_id`,
  ADD COLUMN `effective_model_id` BIGINT NULL AFTER `requested_model_id`,
  ADD COLUMN `model_fallback_reason` VARCHAR(64) NULL AFTER `effective_model_id`,
  ADD COLUMN `capability_version_id` BIGINT NULL AFTER `model_fallback_reason`,
  ADD COLUMN `capability_snapshot_hash` CHAR(64) NULL AFTER `capability_version_id`,
  ADD KEY `idx_candidate_match_capability_version` (`capability_version_id`);
