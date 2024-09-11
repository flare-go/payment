package payment

import (
	"context"

	"github.com/stripe/stripe-go/v79"

	"goflare.io/payment/models"
)

type Payment interface {
	CreateCustomer(ctx context.Context, email, name string) error // Interacts with Stripe
	GetCustomer(ctx context.Context, customerID string) (*models.Customer, error)
	UpdateCustomerBalance(ctx context.Context, customer *models.Customer) error // Interacts with Stripe
	DeleteCustomer(customerID string) error                                     // Interacts with Stripe

	CreateProduct(req models.Product) error // Interacts with Stripe
	GetProductWithActivePrices(ctx context.Context, productID string) (*models.Product, error)
	GetProductWithAllPrices(ctx context.Context, productID string) (*models.Product, error)
	UpdateProduct(product *models.Product) error // Interacts with Stripe
	DeleteProduct(productID string) error        // Interacts with Stripe
	ListProducts(ctx context.Context) ([]*models.Product, error)

	CreatePrice(price models.Price) error // Interacts with Stripe
	DeletePrice(priceID string) error     // Interacts with Stripe

	CreateSubscription(customerID, priceID string) error // Interacts with Stripe
	GetSubscription(ctx context.Context, subscriptionID string) (*models.Subscription, error)
	UpdateSubscription(subscription *models.Subscription) error             // Interacts with Stripe
	CancelSubscription(subscriptionID string, cancelAtPeriodEnd bool) error // Interacts with Stripe
	ListSubscriptions(ctx context.Context, customerID string) ([]*models.Subscription, error)

	CreateInvoice(customerID, subscriptionID string) error // Interacts with Stripe
	GetInvoice(ctx context.Context, invoiceID string) (*models.Invoice, error)
	PayInvoice(invoiceID string) error // Interacts with Stripe
	ListInvoices(ctx context.Context, customerID string) ([]*models.Invoice, error)

	GetPaymentMethod(ctx context.Context, paymentMethodID string) (*models.PaymentMethod, error)
	DeletePaymentMethod(ctx context.Context, paymentMethodID string) error // Interacts with Stripe
	ListPaymentMethods(ctx context.Context, customerID string) ([]*models.PaymentMethod, error)

	CreatePaymentIntent(customerID, paymentMethodStripeID string, amount uint64, currency stripe.Currency) error // Interacts with Stripe
	GetPaymentIntent(ctx context.Context, paymentIntentID string) (*models.PaymentIntent, error)
	ConfirmPaymentIntent(paymentIntentID, paymentMethodID string) error // Interacts with Stripe
	CancelPaymentIntent(paymentIntentID string) error
	ListPaymentIntent(ctx context.Context, limit, offset uint64) ([]*models.PaymentIntent, error) // Interacts with Stripe
	ListPaymentIntentByCustomerID(ctx context.Context, customerID string, limit, offset uint64) ([]*models.PaymentIntent, error)

	CreateRefund(paymentIntentID, reason string, amount uint64) error // Interacts with Stripe
	GetRefund(ctx context.Context, refundID string) (*models.Refund, error)
	UpdateRefund(refundID string, reason string) error // Interacts with Stripe
	ListRefunds(ctx context.Context, chargeID string) ([]*models.Refund, error)

	HandleStripeWebhook(ctx context.Context, payload []byte, signature string) error // Interacts with Stripe

	Close()
}
