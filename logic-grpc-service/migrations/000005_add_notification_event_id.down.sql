-- Down: 000005_add_notification_event_id

ALTER TABLE `notifications`
  DROP KEY uk_notification_once,
  ADD KEY `idx_receiver_biz_type` (`receiver_id`, `receiver_role`, `biz_type`, `biz_id`, `type`),
  DROP KEY uk_notification_event_id,
  DROP COLUMN event_id;
