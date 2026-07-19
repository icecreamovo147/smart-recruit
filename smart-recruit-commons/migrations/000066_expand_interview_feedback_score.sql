ALTER TABLE `interview_feedback`
  MODIFY COLUMN `score` INT DEFAULT NULL COMMENT '综合评分（0-100）',
  ADD CONSTRAINT `chk_interview_feedback_score`
    CHECK (`score` IS NULL OR (`score` BETWEEN 0 AND 100));
