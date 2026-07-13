-- 000051_standardize_event_outbox.sql
-- Adds standard domain-event envelope metadata, retry diagnostics, and retention timestamps.

ALTER TABLE `event_outbox`
  ADD COLUMN `schema_version` VARCHAR(16) NOT NULL DEFAULT '1.0' COMMENT '领域事件信封版本' AFTER `event_id`,
  ADD COLUMN `producer` VARCHAR(128) NOT NULL DEFAULT 'smart-recruit-domain-go.outbox' COMMENT '事件生产者' AFTER `routing_key`,
  ADD COLUMN `idempotency_key` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '消费者幂等键' AFTER `producer`,
  ADD COLUMN `correlation_id` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '请求/流程关联ID' AFTER `idempotency_key`,
  ADD COLUMN `causation_id` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '触发当前事件的命令或事件ID' AFTER `correlation_id`,
  ADD COLUMN `trace_id` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '链路追踪ID' AFTER `causation_id`,
  ADD COLUMN `metadata` JSON NULL COMMENT '安全诊断元数据' AFTER `payload`,
  ADD COLUMN `published_at` DATETIME NULL COMMENT '成功发布到消息队列时间' AFTER `locked_by`,
  ADD COLUMN `dead_lettered_at` DATETIME NULL COMMENT '进入死信状态时间' AFTER `published_at`;

UPDATE `event_outbox`
SET
  `schema_version` = '1.0',
  `producer` = 'smart-recruit-domain-go.outbox',
  `idempotency_key` = CONCAT(`aggregate_type`, ':', `aggregate_id`, ':', `event_type`, ':', `event_id`),
  `metadata` = JSON_OBJECT('routing_key', `routing_key`)
WHERE `idempotency_key` = '';

ALTER TABLE `event_outbox`
  ADD KEY `idx_outbox_idempotency_key` (`idempotency_key`),
  ADD KEY `idx_outbox_published_at` (`status`, `published_at`),
  ADD KEY `idx_outbox_dead_lettered_at` (`status`, `dead_lettered_at`),
  ADD KEY `idx_outbox_status_created` (`status`, `created_at`);
