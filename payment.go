package payment

import (
	"context"

	"github.com/stripe/stripe-go/v79"

	"goflare.io/payment/models"
	"goflare.io/payment/models/enum"
)

type Payment interface {
	CreateCustomer(ctx context.Context, userID uint64, email, name string) (*models.Customer, error) // Interacts with Stripe
	GetCustomer(ctx context.Context, customerID uint64) (*models.Customer, error)
	UpdateCustomer(ctx context.Context, customer *models.Customer) (*models.Customer, error) // Interacts with Stripe
	DeleteCustomer(ctx context.Context, customerID uint64) error                             // Interacts with Stripe

	CreateProduct(ctx context.Context, name, description string, active bool) (*models.Product, error) // Interacts with Stripe
	GetProduct(ctx context.Context, productID uint64) (*models.Product, error)
	UpdateProduct(ctx context.Context, product *models.Product) (*models.Product, error) // Interacts with Stripe
	DeleteProduct(ctx context.Context, productID uint64) error                           // Interacts with Stripe
	ListProducts(ctx context.Context, active bool) ([]*models.Product, error)

	CreatePrice(ctx context.Context, productID uint64, priceType enum.PriceType, currency enum.Currency, unitAmount int64, interval enum.Interval, intervalCount, trialPeriodDays int32) (*models.Price, error) // Interacts with Stripe
	GetPrice(ctx context.Context, priceID uint64) (*models.Price, error)
	UpdatePrice(ctx context.Context, price *models.Price) (*models.Price, error) // Interacts with Stripe
	DeletePrice(ctx context.Context, priceID uint64) error                       // Interacts with Stripe
	ListPrices(ctx context.Context, productID uint64, active bool) ([]*models.Price, error)

	CreateSubscription(ctx context.Context, customerID, priceID uint64) (*models.Subscription, error) // Interacts with Stripe
	GetSubscription(ctx context.Context, subscriptionID uint64) (*models.Subscription, error)
	UpdateSubscription(ctx context.Context, subscription *models.Subscription) (*models.Subscription, error)             // Interacts with Stripe
	CancelSubscription(ctx context.Context, subscriptionID uint64, cancelAtPeriodEnd bool) (*models.Subscription, error) // Interacts with Stripe
	ListSubscriptions(ctx context.Context, customerID uint64) ([]*models.Subscription, error)

	CreateInvoice(ctx context.Context, customerID, subscriptionID uint64) (*models.Invoice, error) // Interacts with Stripe
	GetInvoice(ctx context.Context, invoiceID uint64) (*models.Invoice, error)
	PayInvoice(ctx context.Context, invoiceID uint64) (*models.Invoice, error) // Interacts with Stripe
	ListInvoices(ctx context.Context, customerID uint64) ([]*models.Invoice, error)

	CreatePaymentMethod(ctx context.Context, customerID uint64, paymentMethodType enum.PaymentMethodType, cardDetails *stripe.CardParams) (*models.PaymentMethod, error) // Interacts with Stripe
	GetPaymentMethod(ctx context.Context, paymentMethodID uint64) (*models.PaymentMethod, error)
	UpdatePaymentMethod(ctx context.Context, paymentMethod *models.PaymentMethod) (*models.PaymentMethod, error) // Interacts with Stripe
	DeletePaymentMethod(ctx context.Context, paymentMethodID uint64) error                                       // Interacts with Stripe
	ListPaymentMethods(ctx context.Context, customerID uint64) ([]*models.PaymentMethod, error)

	CreatePaymentIntent(ctx context.Context, customerID uint64, amount int64, currency enum.Currency) (*models.PaymentIntent, error) // Interacts with Stripe
	GetPaymentIntent(ctx context.Context, paymentIntentID uint64) (*models.PaymentIntent, error)
	ConfirmPaymentIntent(ctx context.Context, paymentIntentID, paymentMethodID uint64) (*models.PaymentIntent, error) // Interacts with Stripe
	CancelPaymentIntent(ctx context.Context, paymentIntentID uint64) (*models.PaymentIntent, error)                   // Interacts with Stripe

	HandleStripeWebhook(ctx context.Context, payload []byte, signature string) error // Interacts with Stripe
}
