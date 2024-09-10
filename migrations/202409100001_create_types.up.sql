-- Coupon Duration ENUM
CREATE TYPE coupon_duration AS ENUM (
    'forever',
    'once',
    'repeating'
    );

-- Checkout Session Status ENUM
CREATE TYPE checkout_session_status AS ENUM (
    'complete',
    'expired',
    'open'
    );

-- Checkout Session Mode ENUM
CREATE TYPE checkout_session_mode AS ENUM (
    'payment',
    'setup',
    'subscription'
    );

-- Quote Status ENUM
CREATE TYPE quote_status AS ENUM (
    'accepted',
    'canceled',
    'draft',
    'open'
    );

-- Review Reason ENUM
CREATE TYPE review_reason AS ENUM (
    'approved',
    'disputed',
    'manual',
    'refunded',
    'refunded_as_fraud',
    'redacted',
    'rule'
    );

-- Review Closed Reason ENUM
CREATE TYPE review_closed_reason AS ENUM (
    'approved',
    'disputed',
    'redacted',
    'refunded',
    'refunded_as_fraud'
    );

-- Invoice Status ENUM
CREATE TYPE invoice_status AS ENUM (
    'draft',
    'open',
    'paid',
    'uncollectible',
    'void'
    );

-- Subscription Status ENUM
CREATE TYPE subscription_status AS ENUM (
    'active',
    'canceled',
    'incomplete',
    'incomplete_expired',
    'past_due',
    'paused',
    'trialing',
    'unpaid'
    );

-- Price Recurring Interval ENUM
CREATE TYPE price_recurring_interval AS ENUM (
    'day',
    'month',
    'week',
    'year'
    );

-- Currency ENUM
CREATE TYPE currency AS ENUM (
    'aed', 'afn', 'all', 'amd', 'ang', 'aoa', 'ars', 'aud', 'awg', 'azn',
    'bam', 'bbd', 'bdt', 'bgn', 'bif', 'bmd', 'bnd', 'bob', 'brl', 'bsd',
    'bwp', 'bzd', 'cad', 'cdf', 'chf', 'clp', 'cny', 'cop', 'crc', 'cve',
    'czk', 'djf', 'dkk', 'dop', 'dzd', 'eek', 'egp', 'etb', 'eur', 'fjd',
    'fkp', 'gbp', 'gel', 'gip', 'gmd', 'gnf', 'gtq', 'gyd', 'hkd', 'hnl',
    'hrk', 'htg', 'huf', 'idr', 'ils', 'inr', 'isk', 'jmd', 'jpy', 'kes',
    'kgs', 'khr', 'kmf', 'krw', 'kyd', 'kzt', 'lak', 'lbp', 'lkr', 'lrd',
    'lsl', 'ltl', 'lvl', 'mad', 'mdl', 'mga', 'mkd', 'mnt', 'mop', 'mro',
    'mur', 'mvr', 'mwk', 'mxn', 'myr', 'mzn', 'nad', 'ngn', 'nio', 'nok',
    'npr', 'nzd', 'pab', 'pen', 'pgk', 'php', 'pkr', 'pln', 'pyg', 'qar',
    'ron', 'rsd', 'rub', 'rwf', 'sar', 'sbd', 'scr', 'sek', 'sgd', 'shp',
    'sll', 'sos', 'srd', 'std', 'svc', 'szl', 'thb', 'tjs', 'top', 'try',
    'ttd', 'twd', 'tzs', 'uah', 'ugx', 'usd', 'uyu', 'uzs', 'vef', 'vnd',
    'vuv', 'wst', 'xaf', 'xcd', 'xof', 'xpf', 'yer', 'zar', 'zmw'
    );

-- Price Type ENUM
CREATE TYPE price_type AS ENUM (
    'one_time',
    'recurring'
    );

-- Refund Status ENUM
CREATE TYPE refund_status AS ENUM (
    'canceled',
    'failed',
    'pending',
    'succeeded',
    'requires_action'
    );

-- Refund Reason ENUM
CREATE TYPE refund_reason AS ENUM (
    'duplicate',
    'expired_uncaptured_charge',
    'fraudulent',
    'requested_by_customer'
    );

-- Payment Intent Status ENUM
CREATE TYPE payment_intent_status AS ENUM (
    'canceled',
    'processing',
    'requires_action',
    'requires_capture',
    'requires_confirmation',
    'requires_payment_method',
    'succeeded'
    );

-- Payment Intent Setup Future Usage ENUM
CREATE TYPE payment_intent_setup_future_usage AS ENUM (
    'off_session',
    'on_session'
    );

-- Payment Intent Capture Method ENUM
CREATE TYPE payment_intent_capture_method AS ENUM (
    'automatic',
    'automatic_async',
    'manual'
    );

-- Payment Method Type ENUM
CREATE TYPE payment_method_type AS ENUM (
    'acss_debit',
    'affirm',
    'afterpay_clearpay',
    'alipay',
    'amazon_pay',
    'au_becs_debit',
    'bacs_debit',
    'bancontact',
    'blik',
    'boleto',
    'card',
    'card_present',
    'cashapp',
    'customer_balance',
    'eps',
    'fpx',
    'giropay',
    'grabpay',
    'ideal',
    'interac_present',
    'klarna',
    'konbini',
    'link',
    'mobilepay',
    'multibanco',
    'oxxo',
    'p24',
    'paynow',
    'paypal',
    'pix',
    'promptpay',
    'revolut_pay',
    'sepa_debit',
    'sofort',
    'swish',
    'twint',
    'us_bank_account',
    'wechat_pay',
    'zip'
    );

CREATE TYPE dispute_status AS ENUM (
    'lost',
    'needs_response',
    'under_review',
    'warning_closed',
    'warning_needs_response',
    'warning_under_review',
    'won'
    );

CREATE TYPE dispute_reason AS ENUM (
    'bank_cannot_process',
    'check_returned',
    'credit_not_processed',
    'customer_initiated',
    'debit_not_authorized',
    'duplicate',
    'fraudulent',
    'general',
    'incorrect_account_details',
    'insufficient_funds',
    'product_not_received',
    'product_unacceptable',
    'subscription_canceled',
    'unrecognized'
    );

CREATE TYPE payment_method_card_brand AS ENUM (
    'amex',
    'diners',
    'discover',
    'jcb',
    'mastercard',
    'unionpay',
    'unknown',
    'visa'
    );

CREATE TYPE charge_status AS ENUM (
    'failed',
    'pending',
    'succeeded'
    );

