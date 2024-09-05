package payment

import (
	"context"

	"github.com/stripe/stripe-go/v79"

	"goflare.io/payment/models"
	"goflare.io/payment/models/enum"
)

type Payment interface {
	CreateCustomer(ctx context.Context, userID uint64, email, name string) error // Interacts with Stripe
	GetCustomer(ctx context.Context, customerID uint64) (*models.Customer, error)
	UpdateCustomerBalance(ctx context.Context, customer *models.Customer) error // Interacts with Stripe
	DeleteCustomer(ctx context.Context, customerID uint64) error                // Interacts with Stripe

	CreateProduct(ctx context.Context, req models.Product) error // Interacts with Stripe
	GetProductWithActivePrices(ctx context.Context, productID uint64) (*models.Product, error)
	GetProductWithAllPrices(ctx context.Context, productID uint64) (*models.Product, error)
	UpdateProduct(ctx context.Context, product *models.Product) error // Interacts with Stripe
	DeleteProduct(ctx context.Context, productID uint64) error        // Interacts with Stripe
	ListProducts(ctx context.Context) ([]*models.Product, error)

	CreatePrice(ctx context.Context, price models.Price) error // Interacts with Stripe
	DeletePrice(ctx context.Context, priceID uint64) error     // Interacts with Stripe

	CreateSubscription(ctx context.Context, customerID, priceID uint64) (*models.Subscription, error) // Interacts with Stripe
	GetSubscription(ctx context.Context, subscriptionID uint64) (*models.Subscription, error)
	UpdateSubscription(ctx context.Context, subscription *models.Subscription) (*models.Subscription, error)             // Interacts with Stripe
	CancelSubscription(ctx context.Context, subscriptionID uint64, cancelAtPeriodEnd bool) (*models.Subscription, error) // Interacts with Stripe
	ListSubscriptions(ctx context.Context, customerID uint64) ([]*models.Subscription, error)

	CreateInvoice(ctx context.Context, customerID, subscriptionID uint64) (*models.Invoice, error) // Interacts with Stripe
	GetInvoice(ctx context.Context, invoiceID uint64) (*models.Invoice, error)
	PayInvoice(ctx context.Context, invoiceID uint64) (*models.Invoice, error) // Interacts with Stripe
	ListInvoices(ctx context.Context, customerID uint64) ([]*models.Invoice, error)

	CreatePaymentMethod(ctx context.Context, customerID uint64, paymentMethodType enum.PaymentMethodType, cardDetails *stripe.PaymentMethodCardParams) (*models.PaymentMethod, error) // Interacts with Stripe
	GetPaymentMethod(ctx context.Context, paymentMethodID uint64) (*models.PaymentMethod, error)
	UpdatePaymentMethod(ctx context.Context, paymentMethod *models.PaymentMethod) (*models.PaymentMethod, error) // Interacts with Stripe
	DeletePaymentMethod(ctx context.Context, paymentMethodID uint64) error                                       // Interacts with Stripe
	ListPaymentMethods(ctx context.Context, customerID uint64) ([]*models.PaymentMethod, error)

	CreatePaymentIntent(ctx context.Context, customerID, amount uint64, currency enum.Currency) (*models.PaymentIntent, error) // Interacts with Stripe
	GetPaymentIntent(ctx context.Context, paymentIntentID uint64) (*models.PaymentIntent, error)
	ConfirmPaymentIntent(ctx context.Context, paymentIntentID, paymentMethodID uint64) (*models.PaymentIntent, error) // Interacts with Stripe
	CancelPaymentIntent(ctx context.Context, paymentIntentID uint64) (*models.PaymentIntent, error)                   // Interacts with Stripe

	CreateRefund(ctx context.Context, paymentIntentID uint64, amount float64, reason string) (*models.Refund, error) // Interacts with Stripe
	GetRefund(ctx context.Context, refundID uint64) (*models.Refund, error)
	UpdateRefund(ctx context.Context, refundID uint64, reason string) (*models.Refund, error) // Interacts with Stripe
	ListRefunds(ctx context.Context, paymentIntentID uint64) ([]*models.Refund, error)

	HandleStripeWebhook(ctx context.Context, payload []byte, signature string) error // Interacts with Stripe
}
