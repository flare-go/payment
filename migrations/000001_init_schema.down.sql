-- 刪除索引
DROP INDEX IF EXISTS idx_customers_user_id;
DROP INDEX IF EXISTS idx_customers_stripe_id;
DROP INDEX IF EXISTS idx_products_stripe_id;
DROP INDEX IF EXISTS idx_prices_stripe_id;
DROP INDEX IF EXISTS idx_subscriptions_stripe_id;
DROP INDEX IF EXISTS idx_invoices_stripe_id;
DROP INDEX IF EXISTS idx_payment_methods_stripe_id;
DROP INDEX IF EXISTS idx_payment_intents_stripe_id;
DROP INDEX IF EXISTS idx_invoice_items_invoice_id;

-- 刪除表
DROP TABLE IF EXISTS refunds;
DROP TABLE IF EXISTS payment_intents;
DROP TABLE IF EXISTS payment_methods;
DROP TABLE IF EXISTS invoice_items;
DROP TABLE IF EXISTS invoices;
DROP TABLE IF EXISTS subscriptions;
DROP TABLE IF EXISTS prices;
DROP TABLE IF EXISTS products;
DROP TABLE IF EXISTS customers;

-- 刪除 ENUM 類型
DROP TYPE IF EXISTS refund_status;
DROP TYPE IF EXISTS payment_intent_status;
DROP TYPE IF EXISTS payment_method_type;
DROP TYPE IF EXISTS invoice_status;
DROP TYPE IF EXISTS subscription_status;
DROP TYPE IF EXISTS interval_type;
DROP TYPE IF EXISTS currency;
DROP TYPE IF EXISTS price_type;