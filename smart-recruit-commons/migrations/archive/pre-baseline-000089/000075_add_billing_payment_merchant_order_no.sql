-- Separate the stable billing order number from each Alipay payment attempt.
-- Legacy attempts used billing_orders.order_no as out_trade_no and are backfilled
-- so in-flight sandbox notifications and refunds remain resolvable.

ALTER TABLE `billing_payments`
  ADD COLUMN `merchant_order_no` VARCHAR(64) NULL AFTER `payment_no`;

UPDATE `billing_payments` payment
JOIN `billing_orders` orders ON orders.id = payment.order_id
SET payment.merchant_order_no = orders.order_no
WHERE payment.merchant_order_no IS NULL;

ALTER TABLE `billing_payments`
  MODIFY COLUMN `merchant_order_no` VARCHAR(64) NOT NULL,
  ADD UNIQUE KEY `uk_billing_payments_merchant_order` (`payment_environment`, `merchant_order_no`);
