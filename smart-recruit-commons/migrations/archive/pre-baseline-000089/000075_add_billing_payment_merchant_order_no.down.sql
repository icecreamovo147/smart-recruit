ALTER TABLE `billing_payments`
  DROP INDEX `uk_billing_payments_merchant_order`,
  DROP COLUMN `merchant_order_no`;
