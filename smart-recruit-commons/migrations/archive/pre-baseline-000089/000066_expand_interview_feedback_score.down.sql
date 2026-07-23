ALTER TABLE `interview_feedback`
  DROP CHECK `chk_interview_feedback_score`,
  MODIFY COLUMN `score` INT DEFAULT NULL COMMENT '评分（0-10）';
