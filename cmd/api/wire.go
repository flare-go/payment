//go:build wireinject
// +build wireinject

package main

import (
	"github.com/google/wire"

	"goflare.io/payment"
	"goflare.io/payment/charge"
	"goflare.io/payment/checkout_session"
	"goflare.io/payment/config"
	"goflare.io/payment/coupon"
	"goflare.io/payment/customer"
	"goflare.io/payment/discount"
	"goflare.io/payment/disputes"
	"goflare.io/payment/driver"
	"goflare.io/payment/event"
	"goflare.io/payment/handlers"
	"goflare.io/payment/invoice"
	"goflare.io/payment/payment_intent"
	"goflare.io/payment/payment_link"
	"goflare.io/payment/payment_method"
	"goflare.io/payment/price"
	"goflare.io/payment/product"
	"goflare.io/payment/promotion_code"
	"goflare.io/payment/quote"
	"goflare.io/payment/refund"
	"goflare.io/payment/review"
	"goflare.io/payment/server"
	"goflare.io/payment/subscription"
	"goflare.io/payment/tax_rate"
)

func InitializePaymentService() (*server.Server, error) {

	wire.Build(
		config.ProvideApplicationConfig,
		config.NewLogger,
		config.ProvidePostgresConn,
		config.ProvideEmber,
		config.ProvideIgnite,
		driver.NewTransactionManager,
		customer.NewRepository,
		customer.NewService,
		checkout_session.NewRepository,
		checkout_session.NewService,
		coupon.NewRepository,
		coupon.NewService,
		charge.NewRepository,
		charge.NewService,
		discount.NewRepository,
		discount.NewService,
		disputes.NewRepository,
		disputes.NewService,
		event.NewRepository,
		event.NewService,
		invoice.NewRepository,
		invoice.NewService,
		payment_method.NewRepository,
		payment_method.NewService,
		payment_link.NewRepository,
		payment_link.NewService,
		payment_intent.NewRepository,
		payment_intent.NewService,
		price.NewRepository,
		price.NewService,
		promotion_code.NewRepository,
		promotion_code.NewService,
		product.NewRepository,
		product.NewService,
		review.NewRepository,
		review.NewService,
		refund.NewRepository,
		refund.NewService,
		subscription.NewRepository,
		subscription.NewService,
		tax_rate.NewRepository,
		tax_rate.NewService,
		quote.NewRepository,
		quote.NewService,
		payment.NewStripePayment,
		handlers.NewCustomerHandler,
		handlers.NewProductHandler,
		handlers.NewPriceHandler,
		handlers.NewPaymentIntentHandler,
		handlers.NewWebhookHandler,
		server.NewServer,
	)

	return &server.Server{}, nil
}
