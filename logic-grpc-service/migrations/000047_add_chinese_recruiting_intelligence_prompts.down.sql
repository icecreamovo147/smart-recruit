-- 000047_add_chinese_recruiting_intelligence_prompts.down.sql
-- Description: Remove Chinese DB-backed prompts added by migration 000047.

START TRANSACTION;

DELETE FROM `prompt_templates`
WHERE `agent_type` IN ('resume_profile_extractor', 'job_requirement_extractor', 'candidate_match_evaluator')
  AND `prompt_role` = 'system'
  AND `version` = 2
  AND `name` IN (
    'Resume Profile Extractor System Prompt zh-CN v2',
    'Job Requirement Extractor System Prompt zh-CN v2',
    'Candidate Match Evaluator System Prompt zh-CN v2'
  );

UPDATE `prompt_templates` t
JOIN (
  SELECT `agent_type`, MAX(`version`) AS `version`
  FROM `prompt_templates`
  WHERE `agent_type` IN ('resume_profile_extractor', 'job_requirement_extractor', 'candidate_match_evaluator')
    AND `prompt_role` = 'system'
  GROUP BY `agent_type`
) latest ON latest.`agent_type` = t.`agent_type` AND latest.`version` = t.`version`
SET t.`is_active` = 1
WHERE t.`prompt_role` = 'system';

COMMIT;
