CREATE TABLE customers (
                           id VARCHAR(255) PRIMARY KEY CHECK (id ~ '^[a-z]+_[a-zA-Z0-9]+$'),
                           user_email VARCHAR(255) NOT NULL REFERENCES users(email) UNIQUE,
                           balance BIGINT NOT NULL DEFAULT 0,
                           created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
                           updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);


CREATE TABLE products (
                          id VARCHAR(255) PRIMARY KEY CHECK (id ~ '^[a-z]+_[a-zA-Z0-9]+$'),
                          name VARCHAR(255) NOT NULL CHECK (length(name) >= 2),
                          description TEXT,
                          active BOOLEAN NOT NULL DEFAULT TRUE,
                          metadata JSONB,
                          created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
                          updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE prices (
                        id VARCHAR(255) PRIMARY KEY CHECK (id ~ '^[a-z]+_[a-zA-Z0-9]+$'),
                        product_id VARCHAR(255) NOT NULL REFERENCES products(id) ON DELETE CASCADE,
                        type price_type NOT NULL,
                        currency currency NOT NULL,
                        unit_amount DECIMAL(10, 2) NOT NULL CHECK (unit_amount > 0),
                        recurring_interval price_recurring_interval,
                        recurring_interval_count INTEGER NOT NULL DEFAULT 1,
                        trial_period_days INTEGER NOT NULL DEFAULT 0 CHECK (trial_period_days >= 0),
                        active BOOLEAN NOT NULL DEFAULT TRUE,
                        created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
                        updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE subscriptions (
                               id VARCHAR(255) PRIMARY KEY CHECK (id ~ '^[a-z]+_[a-zA-Z0-9]+$'),
                               customer_id VARCHAR(255) NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
                               price_id VARCHAR(255) NOT NULL REFERENCES prices(id),
                               status subscription_status NOT NULL,
                               current_period_start TIMESTAMP WITH TIME ZONE NOT NULL,
                               current_period_end TIMESTAMP WITH TIME ZONE NOT NULL,
                               canceled_at TIMESTAMP WITH TIME ZONE,
                               cancel_at_period_end BOOLEAN NOT NULL DEFAULT FALSE,
                               trial_start TIMESTAMP WITH TIME ZONE,
                               trial_end TIMESTAMP WITH TIME ZONE,
                               created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
                               updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
                               CHECK (current_period_end > current_period_start),
                               CHECK ((trial_start IS NULL AND trial_end IS NULL) OR (trial_start IS NOT NULL AND trial_end IS NOT NULL AND trial_end > trial_start))
);

CREATE TABLE invoices (
                          id VARCHAR(255) PRIMARY KEY CHECK (id ~ '^[a-z]+_[a-zA-Z0-9]+$'),
                          customer_id VARCHAR(255) NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
                          subscription_id VARCHAR(255) REFERENCES subscriptions(id) ON DELETE SET NULL,
                          status invoice_status NOT NULL,
                          currency currency NOT NULL,
                          amount_due DECIMAL(10, 2) NOT NULL CHECK (amount_due >= 0),
                          amount_paid DECIMAL(10, 2) NOT NULL DEFAULT 0 CHECK (amount_paid >= 0),
                          amount_remaining DECIMAL(10, 2) NOT NULL CHECK (amount_remaining >= 0),
                          due_date TIMESTAMP WITH TIME ZONE,
                          paid_at TIMESTAMP WITH TIME ZONE,
                          created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
                          updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
                          CHECK (amount_due = amount_paid + amount_remaining),
                          CHECK (paid_at IS NULL OR paid_at <= NOW())
);

CREATE TABLE invoice_items (
                               id VARCHAR(255) PRIMARY KEY CHECK (id ~ '^[a-z]+_[a-zA-Z0-9]+$'),
                               invoice_id VARCHAR(255) NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
                               amount DECIMAL(10, 2) NOT NULL,
                               description TEXT,
                               created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
                               updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);


CREATE TABLE payment_methods (
                                 id VARCHAR(255) PRIMARY KEY CHECK (id ~ '^[a-z]+_[a-zA-Z0-9]+$'),
                                 customer_id VARCHAR(255) NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
                                 type payment_method_type NOT NULL,
                                 card_last4 VARCHAR(4) CHECK (card_last4 ~ '^[0-9]{4}$'),
                                 card_brand payment_method_card_brand,
                                 card_exp_month INTEGER CHECK (card_exp_month BETWEEN 1 AND 12),
                                 card_exp_year INTEGER CHECK (card_exp_year >= EXTRACT(YEAR FROM CURRENT_DATE)),
                                 bank_account_last4 VARCHAR(4) CHECK (bank_account_last4 ~ '^[0-9]{4}$'),
                                 bank_account_bank_name VARCHAR(255),
                                 is_default BOOLEAN NOT NULL DEFAULT FALSE,
                                 created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
                                 updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
                                 CHECK ((type = 'card' AND card_last4 IS NOT NULL AND card_brand IS NOT NULL AND card_exp_month IS NOT NULL AND card_exp_year IS NOT NULL) OR
                                        (type = 'us_bank_account' AND bank_account_last4 IS NOT NULL AND bank_account_bank_name IS NOT NULL))
);



CREATE TABLE payment_intents (
                                 id VARCHAR(255) PRIMARY KEY CHECK (id ~ '^[a-z]+_[a-zA-Z0-9]+$'),
                                 customer_id VARCHAR(255),
                                 amount DECIMAL(10, 2) NOT NULL CHECK (amount > 0),
                                 currency currency NOT NULL,
                                 capture_method payment_intent_capture_method NOT NULL,
                                 status payment_intent_status NOT NULL,
                                 payment_method_id VARCHAR(255),
                                 setup_future_usage payment_intent_setup_future_usage,
                                 client_secret VARCHAR(255) NOT NULL,
                                 created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
                                 updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE charges (
                         id VARCHAR(255) PRIMARY KEY CHECK (id ~ '^[a-z]+_[a-zA-Z0-9]+$'),
                         customer_id VARCHAR(255) REFERENCES customers(id),
                         payment_intent_id VARCHAR(255),
                         amount BIGINT NOT NULL,
                         currency currency NOT NULL,
                         status charge_status NOT NULL,
                         paid BOOLEAN NOT NULL,
                         refunded BOOLEAN NOT NULL,
                         failure_code VARCHAR(100),
                         failure_message TEXT,
                         created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
                         updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);


CREATE TABLE refunds (
                         id VARCHAR(255) PRIMARY KEY CHECK (id ~ '^[a-z]+_[a-zA-Z0-9]+$'),
                         charge_id VARCHAR(255) NOT NULL REFERENCES charges(id) ON DELETE CASCADE,
                         amount DECIMAL(10, 2) NOT NULL CHECK (amount > 0),
                         status refund_status NOT NULL,
                         reason refund_reason,
                         created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
                         updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE events (
                        id VARCHAR(255) PRIMARY KEY CHECK (id ~ '^[a-z]+_[a-zA-Z0-9]+$'),
                        type event_type NOT NULL,
                        processed BOOLEAN NOT NULL DEFAULT FALSE,
                        created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
                        updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE disputes (
                          id VARCHAR(255) PRIMARY KEY CHECK (id ~ '^[a-z]+_[a-zA-Z0-9]+$'),
                          charge_id VARCHAR(255) NOT NULL REFERENCES charges(id),
                          amount BIGINT NOT NULL,
                          currency currency NOT NULL,
                          status dispute_status NOT NULL,
                          reason dispute_reason NOT NULL,
                          evidence_due_by TIMESTAMP WITH TIME ZONE NOT NULL,
                          created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
                          updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE coupons (
                         id VARCHAR(255) PRIMARY KEY CHECK (id ~ '^[a-z]+_[a-zA-Z0-9]+$'),
                         name VARCHAR(255) NOT NULL,
                         amount_off BIGINT NOT NULL DEFAULT 0,
                         percent_off DECIMAL(5,2) NOT NULL DEFAULT 0,
                         currency currency NOT NULL,
                         duration coupon_duration NOT NULL,
                         duration_in_months INTEGER NOT NULL DEFAULT 0,
                         max_redemptions INTEGER NOT NULL DEFAULT 0,
                         times_redeemed INTEGER NOT NULL DEFAULT 0,
                         valid BOOLEAN NOT NULL DEFAULT TRUE,
                         created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
                         updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
                         redeem_by TIMESTAMP WITH TIME ZONE
);


CREATE TABLE discounts (
                           id VARCHAR(255) PRIMARY KEY CHECK (id ~ '^[a-z]+_[a-zA-Z0-9]+$'),
                           customer_id VARCHAR(255) NOT NULL REFERENCES customers(id),
                           coupon_id VARCHAR(255) NOT NULL REFERENCES coupons(id),
                           start TIMESTAMP WITH TIME ZONE NOT NULL,
                           "end" TIMESTAMP WITH TIME ZONE,
                           created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
                           updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- 促銷代碼表
CREATE TABLE promotion_codes (
                                 id VARCHAR(255) PRIMARY KEY CHECK (id ~ '^[a-z]+_[a-zA-Z0-9]+$'),
                                 code VARCHAR(255) NOT NULL UNIQUE,
                                 coupon_id VARCHAR(255) NOT NULL REFERENCES coupons(id),
                                 customer_id VARCHAR(255) REFERENCES customers(id),
                                 active BOOLEAN NOT NULL DEFAULT TRUE,
                                 max_redemptions INTEGER,
                                 times_redeemed INTEGER NOT NULL DEFAULT 0,
                                 expires_at TIMESTAMP WITH TIME ZONE,
                                 created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
                                 updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- 結帳會話表
CREATE TABLE checkout_sessions (
                                   id VARCHAR(255) PRIMARY KEY,
                                   customer_id VARCHAR(255) REFERENCES customers(id),
                                   payment_intent_id VARCHAR(255),
                                   status checkout_session_status NOT NULL,
                                   mode checkout_session_mode NOT NULL,
                                   success_url TEXT NOT NULL,
                                   cancel_url TEXT NOT NULL,
                                   amount_total BIGINT NOT NULL,
                                   currency currency NOT NULL,
                                   created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
                                   updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- 報價表
CREATE TABLE quotes (
                        id VARCHAR(255) PRIMARY KEY CHECK (id ~ '^[a-z]+_[a-zA-Z0-9]+$'),
                        customer_id VARCHAR(255) NOT NULL REFERENCES customers(id),
                        status quote_status NOT NULL,
                        amount_total BIGINT NOT NULL,
                        currency currency NOT NULL,
                        valid_until TIMESTAMP WITH TIME ZONE,
                        accepted_at TIMESTAMP WITH TIME ZONE,
                        canceled_at TIMESTAMP WITH TIME ZONE,
                        created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
                        updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- 支付連結表
CREATE TABLE payment_links (
                               id VARCHAR(255) PRIMARY KEY CHECK (id ~ '^[a-z]+_[a-zA-Z0-9]+$'),
                               active BOOLEAN NOT NULL DEFAULT TRUE,
                               url TEXT NOT NULL,
                               amount BIGINT NOT NULL,
                               currency currency NOT NULL,
                               created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
                               updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- 稅率表
CREATE TABLE tax_rates (
                           id VARCHAR(255) PRIMARY KEY CHECK (id ~ '^[a-z]+_[a-zA-Z0-9]+$'),
                           display_name VARCHAR(255) NOT NULL,
                           description TEXT,
                           jurisdiction VARCHAR(255),
                           percentage DECIMAL(5,2) NOT NULL,
                           inclusive BOOLEAN NOT NULL DEFAULT FALSE,
                           active BOOLEAN NOT NULL DEFAULT TRUE,
                           created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
                           updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- 審查表
CREATE TABLE reviews (
                         id VARCHAR(255) PRIMARY KEY CHECK (id ~ '^[a-z]+_[a-zA-Z0-9]+$'),
                         payment_intent_id VARCHAR(255) REFERENCES payment_intents(id),
                         reason review_reason NOT NULL,
                         closed_reason review_closed_reason NOT NULL,
                         status VARCHAR(50) NOT NULL,
                         opened_at TIMESTAMP WITH TIME ZONE NOT NULL,
                         closed_at TIMESTAMP WITH TIME ZONE,
                         created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
                         updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);


-- 創建索引
CREATE INDEX idx_customers_id ON customers(id);
CREATE INDEX idx_products_id ON products(id);
CREATE INDEX idx_prices_id ON prices(id);
CREATE INDEX idx_subscriptions_id ON subscriptions(id);
CREATE INDEX idx_invoices_id ON invoices(id);
CREATE INDEX idx_payment_methods_id ON payment_methods(id);
CREATE INDEX idx_payment_intents_id ON payment_intents(id);
CREATE INDEX idx_invoice_items_invoice_id ON invoice_items(invoice_id);
CREATE INDEX idx_refunds_payment_intent_id ON refunds(charge_id);
CREATE INDEX idx_refunds_id ON refunds(id);
CREATE INDEX idx_events_processed ON events(processed);
CREATE INDEX idx_disputes_id ON disputes(id);
CREATE INDEX idx_disputes_charge_id ON disputes(charge_id);
CREATE INDEX idx_coupons_id ON coupons(id);
CREATE INDEX idx_discounts_customer_id ON discounts(customer_id);
CREATE INDEX idx_discounts_coupon_id ON discounts(coupon_id);
CREATE INDEX idx_promotion_codes_coupon_id ON promotion_codes(coupon_id);
CREATE INDEX idx_promotion_codes_customer_id ON promotion_codes(customer_id);
CREATE INDEX idx_checkout_sessions_customer_id ON checkout_sessions(customer_id);
CREATE INDEX idx_checkout_sessions_payment_intent_id ON checkout_sessions(payment_intent_id);
CREATE INDEX idx_quotes_customer_id ON quotes(customer_id);
CREATE INDEX idx_reviews_payment_intent_id ON reviews(payment_intent_id);
