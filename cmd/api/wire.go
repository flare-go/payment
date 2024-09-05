//go:build wireinject
// +build wireinject

package main

import (
	"github.com/google/wire"
	"goflare.io/payment/handlers"
	"goflare.io/payment/server"

	"goflare.io/payment"
	"goflare.io/payment/config"
	"goflare.io/payment/customer"
	"goflare.io/payment/driver"
	"goflare.io/payment/invoice"
	"goflare.io/payment/payment_intent"
	"goflare.io/payment/payment_method"
	"goflare.io/payment/price"
	"goflare.io/payment/product"
	"goflare.io/payment/refund"
	"goflare.io/payment/subscription"
)

func InitializeAuthService() (*server.Server, error) {

	wire.Build(
		config.ProvideApplicationConfig,
		config.NewLogger,
		config.ProvidePostgresConn,
		config.ProvideEmber,
		config.ProvideIgnite,
		driver.NewTransactionManager,
		customer.NewRepository,
		customer.NewService,
		invoice.NewRepository,
		invoice.NewService,
		payment_method.NewRepository,
		payment_method.NewService,
		payment_intent.NewRepository,
		payment_intent.NewService,
		price.NewRepository,
		price.NewService,
		product.NewRepository,
		product.NewService,
		refund.NewRepository,
		refund.NewService,
		subscription.NewRepository,
		subscription.NewService,
		payment.NewStripePayment,
		handlers.NewCustomerHandler,
		handlers.NewProductHandler,
		handlers.NewPriceHandler,
		server.NewServer,
	)

	return &server.Server{}, nil
}
