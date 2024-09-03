CREATE TYPE price_type AS ENUM ('ONE_TIME', 'RECURRING');
CREATE TYPE currency AS ENUM ('USD', 'EUR', 'JPY', 'TWD');
CREATE TYPE interval_type AS ENUM ('DAY', 'WEEK', 'MONTH', 'YEAR');
CREATE TYPE subscription_status AS ENUM ('ACTIVE', 'PAST_DUE', 'UNPAID', 'CANCELED', 'INCOMPLETE', 'INCOMPLETE_EXPIRED', 'TRIALING');
CREATE TYPE invoice_status AS ENUM ('DRAFT', 'OPEN', 'PAID', 'UNCOLLECTIBLE', 'VOID');
CREATE TYPE payment_method_type AS ENUM ('CARD', 'BANK_ACCOUNT');
CREATE TYPE payment_intent_status AS ENUM ('REQUIRES_PAYMENT_METHOD', 'REQUIRES_CONFIRMATION', 'REQUIRES_ACTION', 'PROCESSING', 'SUCCEEDED', 'CANCELED');

CREATE TABLE customers (
                           id SERIAL PRIMARY KEY,
                           user_id INTEGER NOT NULL REFERENCES users(id) UNIQUE,  -- 確保每個用戶只有一個對應的 customer
                           balance BIGINT NOT NULL DEFAULT 0,
                           stripe_id VARCHAR(255) NOT NULL UNIQUE,
                           created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
                           updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE products (
                          id SERIAL PRIMARY KEY,
                          name VARCHAR(255) NOT NULL CHECK (length(name) >= 2),
                          description TEXT,
                          active BOOLEAN NOT NULL DEFAULT TRUE,
                          metadata JSONB,
                          stripe_id VARCHAR(255) NOT NULL UNIQUE,
                          created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
                          updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE prices (
                        id SERIAL PRIMARY KEY,
                        product_id INTEGER NOT NULL REFERENCES products(id) ON DELETE CASCADE,
                        type price_type NOT NULL,
                        currency currency NOT NULL,
                        unit_amount DECIMAL(10, 2) NOT NULL CHECK (unit_amount > 0),
                        recurring_interval interval_type,
                        recurring_interval_count INTEGER CHECK (recurring_interval_count > 0),
                        trial_period_days INTEGER CHECK (trial_period_days >= 0),
                        active BOOLEAN NOT NULL DEFAULT TRUE,
                        stripe_id VARCHAR(255) NOT NULL UNIQUE,
                        created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
                        updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
                        CHECK ((type = 'RECURRING' AND recurring_interval IS NOT NULL AND recurring_interval_count IS NOT NULL) OR
                               (type = 'ONE_TIME' AND recurring_interval IS NULL AND recurring_interval_count IS NULL))
);

CREATE TABLE subscriptions (
                               id SERIAL PRIMARY KEY,
                               customer_id INTEGER NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
                               price_id INTEGER NOT NULL REFERENCES prices(id),
                               status subscription_status NOT NULL,
                               current_period_start TIMESTAMP WITH TIME ZONE NOT NULL,
                               current_period_end TIMESTAMP WITH TIME ZONE NOT NULL,
                               canceled_at TIMESTAMP WITH TIME ZONE,
                               cancel_at_period_end BOOLEAN NOT NULL DEFAULT FALSE,
                               trial_start TIMESTAMP WITH TIME ZONE,
                               trial_end TIMESTAMP WITH TIME ZONE,
                               stripe_id VARCHAR(255) NOT NULL UNIQUE,
                               created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
                               updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
                               CHECK (current_period_end > current_period_start),
                               CHECK ((trial_start IS NULL AND trial_end IS NULL) OR (trial_start IS NOT NULL AND trial_end IS NOT NULL AND trial_end > trial_start))
);

CREATE TABLE invoices (
                          id SERIAL PRIMARY KEY,
                          customer_id INTEGER NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
                          subscription_id INTEGER REFERENCES subscriptions(id) ON DELETE SET NULL,
                          status invoice_status NOT NULL,
                          currency currency NOT NULL,
                          amount_due BIGINT NOT NULL CHECK (amount_due >= 0),
                          amount_paid BIGINT NOT NULL DEFAULT 0 CHECK (amount_paid >= 0),
                          amount_remaining BIGINT NOT NULL CHECK (amount_remaining >= 0),
                          due_date TIMESTAMP WITH TIME ZONE,
                          paid_at TIMESTAMP WITH TIME ZONE,
                          stripe_id VARCHAR(255) NOT NULL UNIQUE,
                          created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
                          updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
                          CHECK (amount_due = amount_paid + amount_remaining),
                          CHECK (paid_at IS NULL OR paid_at <= NOW())
);

CREATE TABLE invoice_items (
                               id SERIAL PRIMARY KEY,
                               invoice_id INTEGER NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
                               amount BIGINT NOT NULL,
                               description TEXT,
                               created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
                               updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE payment_methods (
                                 id SERIAL PRIMARY KEY,
                                 customer_id INTEGER NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
                                 type payment_method_type NOT NULL,
                                 card_last4 VARCHAR(4) CHECK (card_last4 ~ '^[0-9]{4}$'),
                                 card_brand VARCHAR(50),
                                 card_exp_month INTEGER CHECK (card_exp_month BETWEEN 1 AND 12),
                                 card_exp_year INTEGER CHECK (card_exp_year >= EXTRACT(YEAR FROM CURRENT_DATE)),
                                 bank_account_last4 VARCHAR(4) CHECK (bank_account_last4 ~ '^[0-9]{4}$'),
                                 bank_account_bank_name VARCHAR(255),
                                 is_default BOOLEAN NOT NULL DEFAULT FALSE,
                                 stripe_id VARCHAR(255) NOT NULL UNIQUE,
                                 created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
                                 updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
                                 CHECK ((type = 'CARD' AND card_last4 IS NOT NULL AND card_brand IS NOT NULL AND card_exp_month IS NOT NULL AND card_exp_year IS NOT NULL) OR
                                        (type = 'BANK_ACCOUNT' AND bank_account_last4 IS NOT NULL AND bank_account_bank_name IS NOT NULL))
);

CREATE TABLE payment_intents (
                                 id SERIAL PRIMARY KEY,
                                 customer_id INTEGER NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
                                 amount BIGINT NOT NULL CHECK (amount > 0),
                                 currency currency NOT NULL,
                                 status payment_intent_status NOT NULL,
                                 payment_method_id INTEGER REFERENCES payment_methods(id) ON DELETE SET NULL,
                                 setup_future_usage VARCHAR(50),
                                 stripe_id VARCHAR(255) NOT NULL UNIQUE,
                                 client_secret VARCHAR(255) NOT NULL,
                                 created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
                                 updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- 創建索引
CREATE INDEX idx_customers_user_id ON customers(user_id);
CREATE INDEX idx_customers_stripe_id ON customers(stripe_id);
CREATE INDEX idx_products_stripe_id ON products(stripe_id);
CREATE INDEX idx_prices_stripe_id ON prices(stripe_id);
CREATE INDEX idx_subscriptions_stripe_id ON subscriptions(stripe_id);
CREATE INDEX idx_invoices_stripe_id ON invoices(stripe_id);
CREATE INDEX idx_payment_methods_stripe_id ON payment_methods(stripe_id);
CREATE INDEX idx_payment_intents_stripe_id ON payment_intents(stripe_id);
CREATE INDEX idx_invoice_items_invoice_id ON invoice_items(invoice_id);
