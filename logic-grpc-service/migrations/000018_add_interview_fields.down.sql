-- Down: 000018_add_interview_fields

DROP TABLE IF EXISTS `interview_feedback`;

ALTER TABLE `interview_schedules`
  DROP COLUMN `cancel_reason`,
  DROP COLUMN `internal_note`,
  DROP COLUMN `candidate_note`,
  DROP COLUMN `duration_minutes`,
  DROP COLUMN `location`,
  DROP COLUMN `meeting_url`,
  DROP COLUMN `mode`,
  DROP COLUMN `title`;
