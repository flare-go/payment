// Code generated by Wire. DO NOT EDIT.

//go:generate go run github.com/google/wire/cmd/wire
//go:build !wireinject
// +build !wireinject

package main

import (
	"goflare.io/payment"
	"goflare.io/payment/config"
	"goflare.io/payment/customer"
	"goflare.io/payment/driver"
	"goflare.io/payment/handlers"
	"goflare.io/payment/invoice"
	"goflare.io/payment/payment_intent"
	"goflare.io/payment/payment_method"
	"goflare.io/payment/price"
	"goflare.io/payment/product"
	"goflare.io/payment/refund"
	"goflare.io/payment/server"
	"goflare.io/payment/subscription"
)

// Injectors from wire.go:

func InitializeAuthService() (*server.Server, error) {
	configConfig, err := config.ProvideApplicationConfig()
	if err != nil {
		return nil, err
	}
	postgresPool, err := config.ProvidePostgresConn(configConfig)
	if err != nil {
		return nil, err
	}
	logger := config.NewLogger()
	multiCache, err := config.ProvideEmber(configConfig)
	if err != nil {
		return nil, err
	}
	manager := config.ProvideIgnite()
	repository, err := customer.NewRepository(postgresPool, logger, multiCache, manager)
	if err != nil {
		return nil, err
	}
	transactionManager := driver.NewTransactionManager(postgresPool, logger)
	service := customer.NewService(repository, transactionManager, logger)
	productRepository, err := product.NewRepository(postgresPool, logger, multiCache, manager)
	if err != nil {
		return nil, err
	}
	productService := product.NewService(productRepository, transactionManager, logger)
	priceRepository, err := price.NewRepository(postgresPool, logger, multiCache, manager)
	if err != nil {
		return nil, err
	}
	priceService := price.NewService(priceRepository, transactionManager, logger)
	subscriptionRepository, err := subscription.NewRepository(postgresPool, logger, multiCache, manager)
	if err != nil {
		return nil, err
	}
	subscriptionService := subscription.NewService(subscriptionRepository, transactionManager, logger)
	invoiceRepository, err := invoice.NewRepository(postgresPool, logger, multiCache, manager)
	if err != nil {
		return nil, err
	}
	invoiceService := invoice.NewService(invoiceRepository, transactionManager, logger)
	payment_methodRepository, err := payment_method.NewRepository(postgresPool, logger, multiCache, manager)
	if err != nil {
		return nil, err
	}
	payment_methodService := payment_method.NewService(payment_methodRepository, transactionManager, logger)
	payment_intentRepository, err := payment_intent.NewRepository(postgresPool, logger, multiCache, manager)
	if err != nil {
		return nil, err
	}
	payment_intentService := payment_intent.NewService(payment_intentRepository, transactionManager, logger)
	refundRepository, err := refund.NewRepository(postgresPool, logger, multiCache, manager)
	if err != nil {
		return nil, err
	}
	refundService := refund.NewService(refundRepository, transactionManager, logger)
	paymentPayment := payment.NewStripePayment(configConfig, service, productService, priceService, subscriptionService, invoiceService, payment_methodService, payment_intentService, refundService)
	customerHandler := handlers.NewCustomerHandler(paymentPayment)
	productHandler := handlers.NewProductHandler(paymentPayment, logger)
	priceHandler := handlers.NewPriceHandler(paymentPayment, logger)
	serverServer := server.NewServer(customerHandler, productHandler, priceHandler)
	return serverServer, nil
}
