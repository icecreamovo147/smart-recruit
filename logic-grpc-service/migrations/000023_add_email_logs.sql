-- 000023_add_email_logs.sql
-- Email notification audit log with idempotency via event_id.

CREATE TABLE IF NOT EXISTS email_logs (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    event_id    VARCHAR(64) NOT NULL COMMENT '来源 outbox 事件ID，用于幂等去重',
    user_id     BIGINT NOT NULL COMMENT '接收用户ID',
    email       VARCHAR(128) NOT NULL COMMENT '接收邮箱地址',
    type        VARCHAR(64) NOT NULL COMMENT '通知类型：interview_scheduled / offer_sent 等',
    subject     VARCHAR(256) NOT NULL COMMENT '邮件标题',
    status      VARCHAR(32) NOT NULL DEFAULT 'sent' COMMENT '状态：sent / failed / skipped',
    error_msg   TEXT NULL COMMENT '发送失败原因',
    sent_at     DATETIME NOT NULL COMMENT '发送时间',
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_email_event_id (event_id),
    INDEX idx_email_user_id (user_id),
    INDEX idx_email_type_status (type, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='邮件发送记录表';
