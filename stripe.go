package payment

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/stripe/stripe-go/v79/webhook"
	"strconv"
	"time"

	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/client"

	"goflare.io/payment/customer"
	"goflare.io/payment/invoice"
	"goflare.io/payment/models"
	"goflare.io/payment/models/enum"
	"goflare.io/payment/payment_intent"
	"goflare.io/payment/payment_method"
	"goflare.io/payment/price"
	"goflare.io/payment/product"
	"goflare.io/payment/subscription"
)

type StripePayment struct {
	client               *client.API
	customerService      customer.Service
	productService       product.Service
	priceService         price.Service
	subscriptionService  subscription.Service
	invoiceService       invoice.Service
	paymentMethodService payment_method.Service
	paymentIntentService payment_intent.Service
}

func NewStripePayment(apiKey string,
	cs customer.Service,
	ps product.Service,
	prs price.Service,
	ss subscription.Service,
	is invoice.Service,
	pms payment_method.Service,
	pis payment_intent.Service) Payment {
	return &StripePayment{
		client:               client.New(apiKey, nil),
		customerService:      cs,
		productService:       ps,
		priceService:         prs,
		subscriptionService:  ss,
		invoiceService:       is,
		paymentMethodService: pms,
		paymentIntentService: pis,
	}
}

// CreateCustomer creates a new customer in Stripe and in the local database
func (sp *StripePayment) CreateCustomer(ctx context.Context, userID uint64, email, name string) (*models.Customer, error) {
	params := &stripe.CustomerParams{
		Email: stripe.String(email),
		Name:  stripe.String(name),
		Metadata: map[string]string{
			"user_id": strconv.FormatUint(userID, 10),
		},
	}
	stripeCustomer, err := sp.client.Customers.New(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create Stripe customer: %w", err)
	}

	customerModel := &models.Customer{
		UserID:   userID,
		Email:    email,
		Name:     name,
		StripeID: stripeCustomer.ID,
	}
	if err = sp.customerService.Create(ctx, customerModel); err != nil {
		return nil, fmt.Errorf("failed to create local customer record: %w", err)
	}

	return customerModel, nil
}

// GetCustomer retrieves a customer from the local database
func (sp *StripePayment) GetCustomer(ctx context.Context, customerID uint64) (*models.Customer, error) {
	return sp.customerService.GetByID(ctx, customerID)
}

// UpdateCustomer updates a customer in Stripe and in the local database
func (sp *StripePayment) UpdateCustomer(ctx context.Context, customer *models.Customer) (*models.Customer, error) {
	params := &stripe.CustomerParams{
		Email: stripe.String(customer.Email),
		Name:  stripe.String(customer.Name),
	}
	_, err := sp.client.Customers.Update(customer.StripeID, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update Stripe customer: %w", err)
	}

	err = sp.customerService.Update(ctx, customer)
	if err != nil {
		return nil, fmt.Errorf("failed to update local customer record: %w", err)
	}

	return customer, nil
}

// DeleteCustomer deletes a customer from Stripe and from the local database
func (sp *StripePayment) DeleteCustomer(ctx context.Context, customerID uint64) error {
	customerModel, err := sp.customerService.GetByID(ctx, customerID)
	if err != nil {
		return fmt.Errorf("failed to get customer: %w", err)
	}

	if _, err = sp.client.Customers.Del(customerModel.StripeID, nil); err != nil {
		return fmt.Errorf("failed to delete Stripe customer: %w", err)
	}

	if err = sp.customerService.Delete(ctx, customerID); err != nil {
		return fmt.Errorf("failed to delete local customer record: %w", err)
	}

	return nil
}

// CreateProduct creates a new product in Stripe and in the local database
func (sp *StripePayment) CreateProduct(ctx context.Context, name, description string, active bool) (*models.Product, error) {
	params := &stripe.ProductParams{
		Name:        stripe.String(name),
		Description: stripe.String(description),
		Active:      stripe.Bool(active),
	}
	stripeProduct, err := sp.client.Products.New(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create Stripe product: %w", err)
	}

	productModel := &models.Product{
		Name:        name,
		Description: description,
		Active:      active,
		StripeID:    stripeProduct.ID,
	}

	if err = sp.productService.Create(ctx, productModel); err != nil {
		return nil, fmt.Errorf("failed to create local product record: %w", err)
	}

	return productModel, nil
}

// GetProduct retrieves a product from the local database
func (sp *StripePayment) GetProduct(ctx context.Context, productID uint64) (*models.Product, error) {
	return sp.productService.GetByID(ctx, productID)
}

// UpdateProduct updates a product in Stripe and in the local database
func (sp *StripePayment) UpdateProduct(ctx context.Context, product *models.Product) (*models.Product, error) {
	params := &stripe.ProductParams{
		Name:        stripe.String(product.Name),
		Description: stripe.String(product.Description),
		Active:      stripe.Bool(product.Active),
	}
	_, err := sp.client.Products.Update(product.StripeID, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update Stripe product: %w", err)
	}

	err = sp.productService.Update(ctx, product)
	if err != nil {
		return nil, fmt.Errorf("failed to update local product record: %w", err)
	}

	return product, nil
}

// DeleteProduct deletes a product from Stripe and from the local database
func (sp *StripePayment) DeleteProduct(ctx context.Context, productID uint64) error {
	productModel, err := sp.productService.GetByID(ctx, productID)
	if err != nil {
		return fmt.Errorf("failed to get product: %w", err)
	}

	if _, err = sp.client.Products.Del(productModel.StripeID, nil); err != nil {
		return fmt.Errorf("failed to delete Stripe product: %w", err)
	}

	if err = sp.productService.Delete(ctx, productID); err != nil {
		return fmt.Errorf("failed to delete local product record: %w", err)
	}

	return nil
}

// ListProducts lists all products from the local database
func (sp *StripePayment) ListProducts(ctx context.Context, active bool) ([]*models.Product, error) {
	return sp.productService.List(ctx, 1000, 0, active) // Assuming a large limit, you might want to implement pagination
}

// CreatePrice creates a new price in Stripe and in the local database
func (sp *StripePayment) CreatePrice(ctx context.Context, productID uint64, priceType enum.PriceType, currency enum.Currency, unitAmount float64, interval enum.Interval, intervalCount, trialPeriodDays int32) (*models.Price, error) {
	productModel, err := sp.productService.GetByID(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("failed to get product: %w", err)
	}

	params := &stripe.PriceParams{
		Product:    stripe.String(productModel.StripeID),
		Currency:   stripe.String(string(currency)),
		UnitAmount: stripe.Int64(int64(unitAmount)),
	}

	if priceType == enum.PriceTypeRecurring {
		params.Recurring = &stripe.PriceRecurringParams{
			Interval:        stripe.String(string(interval)),
			IntervalCount:   stripe.Int64(int64(intervalCount)),
			TrialPeriodDays: stripe.Int64(int64(trialPeriodDays)),
		}
	}

	stripePrice, err := sp.client.Prices.New(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create Stripe price: %w", err)
	}

	priceModel := &models.Price{
		ProductID:              productID,
		Type:                   priceType,
		Currency:               currency,
		UnitAmount:             unitAmount,
		RecurringInterval:      interval,
		RecurringIntervalCount: intervalCount,
		TrialPeriodDays:        trialPeriodDays,
		StripeID:               stripePrice.ID,
	}
	if err = sp.priceService.Create(ctx, priceModel); err != nil {
		return nil, fmt.Errorf("failed to create local price record: %w", err)
	}

	return priceModel, nil
}

// GetPrice retrieves a price from the local database
func (sp *StripePayment) GetPrice(ctx context.Context, priceID uint64) (*models.Price, error) {
	return sp.priceService.GetByID(ctx, priceID)
}

// UpdatePrice updates a price in Stripe and in the local database
func (sp *StripePayment) UpdatePrice(ctx context.Context, price *models.Price) (*models.Price, error) {
	params := &stripe.PriceParams{
		Active: stripe.Bool(price.Active),
		// Note: Most fields of a Price cannot be updated after creation in Stripe
		// We're only updating the 'active' status here
	}
	_, err := sp.client.Prices.Update(price.StripeID, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update Stripe price: %w", err)
	}

	err = sp.priceService.Update(ctx, price)
	if err != nil {
		return nil, fmt.Errorf("failed to update local price record: %w", err)
	}

	return price, nil
}

// DeletePrice deletes a price from Stripe and from the local database
func (sp *StripePayment) DeletePrice(ctx context.Context, priceID uint64) error {
	priceModel, err := sp.priceService.GetByID(ctx, priceID)
	if err != nil {
		return fmt.Errorf("failed to get price: %w", err)
	}

	// In Stripe, you can't delete prices, you can only deactivate them
	_, err = sp.client.Prices.Update(priceModel.StripeID, &stripe.PriceParams{
		Active: stripe.Bool(false),
	})
	if err != nil {
		return fmt.Errorf("failed to deactivate Stripe price: %w", err)
	}

	err = sp.priceService.Delete(ctx, priceID)
	if err != nil {
		return fmt.Errorf("failed to delete local price record: %w", err)
	}

	return nil
}

// ListPrices lists all prices for a product from the local database
func (sp *StripePayment) ListPrices(ctx context.Context, productID uint64, active bool) ([]*models.Price, error) {
	return sp.priceService.List(ctx, productID, 1000, 0, active)
	// Assuming a large limit,
	// you might want to implement pagination
}

// CreateSubscription creates a new subscription in Stripe and in the local database
func (sp *StripePayment) CreateSubscription(ctx context.Context, customerID, priceID uint64) (*models.Subscription, error) {
	customerModel, err := sp.customerService.GetByID(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer: %w", err)
	}

	priceModel, err := sp.priceService.GetByID(ctx, priceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get price: %w", err)
	}

	params := &stripe.SubscriptionParams{
		Customer: stripe.String(customerModel.StripeID),
		Items: []*stripe.SubscriptionItemsParams{
			{
				Price: stripe.String(priceModel.StripeID),
			},
		},
	}

	stripeSubscription, err := sp.client.Subscriptions.New(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create Stripe subscription: %w", err)
	}

	subscriptionModel := &models.Subscription{
		CustomerID:         customerID,
		PriceID:            priceID,
		Status:             enum.SubscriptionStatus(stripeSubscription.Status),
		CurrentPeriodStart: time.Unix(stripeSubscription.CurrentPeriodStart, 0),
		CurrentPeriodEnd:   time.Unix(stripeSubscription.CurrentPeriodEnd, 0),
		CancelAtPeriodEnd:  stripeSubscription.CancelAtPeriodEnd,
		StripeID:           stripeSubscription.ID,
	}

	err = sp.subscriptionService.Create(ctx, subscriptionModel)
	if err != nil {
		return nil, fmt.Errorf("failed to create local subscription record: %w", err)
	}

	return subscriptionModel, nil
}

// GetSubscription retrieves a subscription from the local database
func (sp *StripePayment) GetSubscription(ctx context.Context, subscriptionID uint64) (*models.Subscription, error) {
	return sp.subscriptionService.GetByID(ctx, subscriptionID)
}

// UpdateSubscription updates a subscription in Stripe and in the local database
func (sp *StripePayment) UpdateSubscription(ctx context.Context, subscription *models.Subscription) (*models.Subscription, error) {
	params := &stripe.SubscriptionParams{
		CancelAtPeriodEnd: stripe.Bool(subscription.CancelAtPeriodEnd),
	}
	stripeSubscription, err := sp.client.Subscriptions.Update(subscription.StripeID, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update Stripe subscription: %w", err)
	}

	subscription.Status = enum.SubscriptionStatus(stripeSubscription.Status)
	subscription.CurrentPeriodStart = time.Unix(stripeSubscription.CurrentPeriodStart, 0)
	subscription.CurrentPeriodEnd = time.Unix(stripeSubscription.CurrentPeriodEnd, 0)

	err = sp.subscriptionService.Update(ctx, subscription)
	if err != nil {
		return nil, fmt.Errorf("failed to update local subscription record: %w", err)
	}

	return subscription, nil
}

// CancelSubscription cancels a subscription in Stripe and updates the local database
func (sp *StripePayment) CancelSubscription(ctx context.Context, subscriptionID uint64, cancelAtPeriodEnd bool) (*models.Subscription, error) {
	subscriptionModel, err := sp.subscriptionService.GetByID(ctx, subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	params := &stripe.SubscriptionParams{
		CancelAtPeriodEnd: stripe.Bool(cancelAtPeriodEnd),
	}

	stripeSubscription, err := sp.client.Subscriptions.Update(subscriptionModel.StripeID, params)
	if err != nil {
		return nil, fmt.Errorf("failed to cancel Stripe subscription: %w", err)
	}

	now := time.Now()
	subscriptionModel.Status = enum.SubscriptionStatus(stripeSubscription.Status)
	subscriptionModel.CancelAtPeriodEnd = stripeSubscription.CancelAtPeriodEnd
	if !cancelAtPeriodEnd {
		subscriptionModel.CanceledAt = &now
	}

	err = sp.subscriptionService.Update(ctx, subscriptionModel)
	if err != nil {
		return nil, fmt.Errorf("failed to update local subscription record: %w", err)
	}

	return subscriptionModel, nil
}

// ListSubscriptions lists all subscriptions for a customer from the local database
func (sp *StripePayment) ListSubscriptions(ctx context.Context, customerID uint64) ([]*models.Subscription, error) {
	return sp.subscriptionService.List(ctx, customerID, 1000, 0)
	// Assuming a large limit, you might want to implement pagination
}

// CreateInvoice creates a new invoice in Stripe and in the local database
func (sp *StripePayment) CreateInvoice(ctx context.Context, customerID, subscriptionID uint64) (*models.Invoice, error) {
	customerModel, err := sp.customerService.GetByID(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer: %w", err)
	}

	params := &stripe.InvoiceParams{
		Customer: stripe.String(customerModel.StripeID),
	}

	if subscriptionID != 0 {
		subscriptionModel, err := sp.subscriptionService.GetByID(ctx, subscriptionID)
		if err != nil {
			return nil, fmt.Errorf("failed to get subscription: %w", err)
		}
		params.Subscription = stripe.String(subscriptionModel.StripeID)
	}

	stripeInvoice, err := sp.client.Invoices.New(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create Stripe invoice: %w", err)
	}

	invoiceModel := &models.Invoice{
		CustomerID:      customerID,
		SubscriptionID:  &subscriptionID,
		Status:          enum.InvoiceStatus(stripeInvoice.Status),
		Currency:        enum.Currency(stripeInvoice.Currency),
		AmountDue:       uint64(stripeInvoice.AmountDue),
		AmountPaid:      uint64(stripeInvoice.AmountPaid),
		AmountRemaining: uint64(stripeInvoice.AmountRemaining),
		DueDate:         time.Unix(stripeInvoice.DueDate, 0),
		StripeID:        stripeInvoice.ID,
	}

	err = sp.invoiceService.Create(ctx, invoiceModel)
	if err != nil {
		return nil, fmt.Errorf("failed to create local invoice record: %w", err)
	}

	return invoiceModel, nil
}

// GetInvoice retrieves an invoice from the local database
func (sp *StripePayment) GetInvoice(ctx context.Context, invoiceID uint64) (*models.Invoice, error) {
	return sp.invoiceService.GetByID(ctx, invoiceID)
}

// PayInvoice pays an invoice in Stripe and updates the local database
func (sp *StripePayment) PayInvoice(ctx context.Context, invoiceID uint64) (*models.Invoice, error) {
	invoiceModel, err := sp.invoiceService.GetByID(ctx, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get invoice: %w", err)
	}

	stripeInvoice, err := sp.client.Invoices.Pay(invoiceModel.StripeID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to pay Stripe invoice: %w", err)
	}

	invoiceModel.Status = enum.InvoiceStatus(stripeInvoice.Status)
	invoiceModel.AmountPaid = uint64(stripeInvoice.AmountPaid)
	invoiceModel.AmountRemaining = uint64(stripeInvoice.AmountRemaining)
	if stripeInvoice.Status == stripe.InvoiceStatusPaid {
		invoiceModel.PaidAt = time.Now()
	}

	err = sp.invoiceService.Update(ctx, invoiceModel)
	if err != nil {
		return nil, fmt.Errorf("failed to update local invoice record: %w", err)
	}

	return invoiceModel, nil
}

// ListInvoices lists all invoices for a customer from the local database
func (sp *StripePayment) ListInvoices(ctx context.Context, customerID uint64) ([]*models.Invoice, error) {
	return sp.invoiceService.List(ctx, customerID, 1000, 0)
	// Assuming a large limit, you might want to implement pagination
}

// CreatePaymentMethod creates a new payment method in Stripe and in the local database
func (sp *StripePayment) CreatePaymentMethod(ctx context.Context, customerID uint64, paymentMethodType enum.PaymentMethodType, cardDetails *stripe.PaymentMethodCardParams) (*models.PaymentMethod, error) {
	customerModel, err := sp.customerService.GetByID(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer: %w", err)
	}

	params := &stripe.PaymentMethodParams{
		Type: stripe.String(string(paymentMethodType)),
		Card: cardDetails,
	}

	stripePaymentMethod, err := sp.client.PaymentMethods.New(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create Stripe payment method: %w", err)
	}

	// Attach the payment method to the customer
	_, err = sp.client.PaymentMethods.Attach(stripePaymentMethod.ID, &stripe.PaymentMethodAttachParams{
		Customer: stripe.String(customerModel.StripeID),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to attach payment method to customer: %w", err)
	}

	paymentMethod := &models.PaymentMethod{
		CustomerID:   customerID,
		Type:         paymentMethodType,
		CardLast4:    stripePaymentMethod.Card.Last4,
		CardBrand:    string(stripePaymentMethod.Card.Brand),
		CardExpMonth: int32(stripePaymentMethod.Card.ExpMonth),
		CardExpYear:  int32(stripePaymentMethod.Card.ExpYear),
		StripeID:     stripePaymentMethod.ID,
	}

	if err = sp.paymentMethodService.Create(ctx, paymentMethod); err != nil {
		return nil, fmt.Errorf("failed to create local payment method record: %w", err)
	}

	return paymentMethod, nil
}

// GetPaymentMethod retrieves a payment method from the local database
func (sp *StripePayment) GetPaymentMethod(ctx context.Context, paymentMethodID uint64) (*models.PaymentMethod, error) {
	return sp.paymentMethodService.GetByID(ctx, paymentMethodID)
}

// UpdatePaymentMethod updates a payment method in Stripe and in the local database
func (sp *StripePayment) UpdatePaymentMethod(ctx context.Context, paymentMethod *models.PaymentMethod) (*models.PaymentMethod, error) {
	// Note: Stripe doesn't allow updating most payment method details.
	// We'll just update our local record.
	err := sp.paymentMethodService.Update(ctx, paymentMethod)
	if err != nil {
		return nil, fmt.Errorf("failed to update local payment method record: %w", err)
	}

	return paymentMethod, nil
}

// DeletePaymentMethod deletes a payment method from Stripe and from the local database
func (sp *StripePayment) DeletePaymentMethod(ctx context.Context, paymentMethodID uint64) error {
	paymentMethod, err := sp.paymentMethodService.GetByID(ctx, paymentMethodID)
	if err != nil {
		return fmt.Errorf("failed to get payment method: %w", err)
	}

	_, err = sp.client.PaymentMethods.Detach(paymentMethod.StripeID, nil)
	if err != nil {
		return fmt.Errorf("failed to detach Stripe payment method: %w", err)
	}

	err = sp.paymentMethodService.Delete(ctx, paymentMethodID)
	if err != nil {
		return fmt.Errorf("failed to delete local payment method record: %w", err)
	}

	return nil
}

// ListPaymentMethods lists all payment methods for a customer from the local database
func (sp *StripePayment) ListPaymentMethods(ctx context.Context, customerID uint64) ([]*models.PaymentMethod, error) {
	return sp.paymentMethodService.List(ctx, customerID, 1000, 0)
	// Assuming a large limit, you might want to implement pagination
}

// CreatePaymentIntent creates a new payment intent in Stripe and in the local database
func (sp *StripePayment) CreatePaymentIntent(ctx context.Context, customerID, amount uint64, currency enum.Currency) (*models.PaymentIntent, error) {
	customerModel, err := sp.customerService.GetByID(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer: %w", err)
	}

	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(int64(amount)),
		Currency: stripe.String(string(currency)),
		Customer: stripe.String(customerModel.StripeID),
	}

	stripePaymentIntent, err := sp.client.PaymentIntents.New(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create Stripe payment intent: %w", err)
	}

	paymentIntent := &models.PaymentIntent{
		CustomerID:   customerID,
		Amount:       amount,
		Currency:     currency,
		Status:       enum.PaymentIntentStatus(stripePaymentIntent.Status),
		ClientSecret: stripePaymentIntent.ClientSecret,
		StripeID:     stripePaymentIntent.ID,
	}

	err = sp.paymentIntentService.Create(ctx, paymentIntent)
	if err != nil {
		return nil, fmt.Errorf("failed to create local payment intent record: %w", err)
	}

	return paymentIntent, nil
}

// GetPaymentIntent retrieves a payment intent from the local database
func (sp *StripePayment) GetPaymentIntent(ctx context.Context, paymentIntentID uint64) (*models.PaymentIntent, error) {
	return sp.paymentIntentService.GetByID(ctx, paymentIntentID)
}

// ConfirmPaymentIntent confirms a payment intent in Stripe and updates the local database
func (sp *StripePayment) ConfirmPaymentIntent(ctx context.Context, paymentIntentID, paymentMethodID uint64) (*models.PaymentIntent, error) {
	paymentIntent, err := sp.paymentIntentService.GetByID(ctx, paymentIntentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment intent: %w", err)
	}

	paymentMethod, err := sp.paymentMethodService.GetByID(ctx, paymentMethodID)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment method: %w", err)
	}

	params := &stripe.PaymentIntentConfirmParams{
		PaymentMethod: stripe.String(paymentMethod.StripeID),
	}

	stripePaymentIntent, err := sp.client.PaymentIntents.Confirm(paymentIntent.StripeID, params)
	if err != nil {
		return nil, fmt.Errorf("failed to confirm Stripe payment intent: %w", err)
	}

	paymentIntent.Status = enum.PaymentIntentStatus(stripePaymentIntent.Status)
	paymentIntent.PaymentMethodID = &paymentMethodID

	err = sp.paymentIntentService.Update(ctx, paymentIntent)
	if err != nil {
		return nil, fmt.Errorf("failed to update local payment intent record: %w", err)
	}

	return paymentIntent, nil
}

// CancelPaymentIntent cancels a payment intent in Stripe and updates the local database
func (sp *StripePayment) CancelPaymentIntent(ctx context.Context, paymentIntentID uint64) (*models.PaymentIntent, error) {
	paymentIntent, err := sp.paymentIntentService.GetByID(ctx, paymentIntentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment intent: %w", err)
	}

	stripePaymentIntent, err := sp.client.PaymentIntents.Cancel(paymentIntent.StripeID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to cancel Stripe payment intent: %w", err)
	}

	paymentIntent.Status = enum.PaymentIntentStatus(stripePaymentIntent.Status)

	err = sp.paymentIntentService.Update(ctx, paymentIntent)
	if err != nil {
		return nil, fmt.Errorf("failed to update local payment intent record: %w", err)
	}

	return paymentIntent, nil
}

// HandleStripeWebhook handles Stripe webhook events
func (sp *StripePayment) HandleStripeWebhook(ctx context.Context, payload []byte, signature string) error {
	event, err := webhook.ConstructEvent(payload, signature, "")
	if err != nil {
		return fmt.Errorf("failed to verify webhook signature: %w", err)
	}

	switch event.Type {
	case "customer.subscription.created", "customer.subscription.updated", "customer.subscription.deleted":
		var subscriptionStripeModel stripe.Subscription
		err = json.Unmarshal(event.Data.Raw, &subscriptionStripeModel)
		if err != nil {
			return fmt.Errorf("failed to unmarshal subscription data: %w", err)
		}
		return sp.handleSubscriptionEvent(ctx, &subscriptionStripeModel, event.Type)

	case "invoice.paid", "invoice.payment_failed":
		var invoiceStripeModel stripe.Invoice
		err := json.Unmarshal(event.Data.Raw, &invoiceStripeModel)
		if err != nil {
			return fmt.Errorf("failed to unmarshal invoice data: %w", err)
		}
		return sp.handleInvoiceEvent(ctx, &invoiceStripeModel, event.Type)

	// Add more event types as needed

	default:
		return fmt.Errorf("unhandled event type: %s", event.Type)
	}
}

// handleSubscriptionEvent handles subscription-related webhook events
func (sp *StripePayment) handleSubscriptionEvent(ctx context.Context, stripeSubscription *stripe.Subscription, eventType stripe.EventType) error {
	// Find the local subscription
	subscriptions, err := sp.subscriptionService.ListByStripeID(ctx, stripeSubscription.ID)
	if err != nil || len(subscriptions) == 0 {
		return fmt.Errorf("failed to find local subscription for Stripe ID %s: %w", stripeSubscription.ID, err)
	}
	subscriptionModel := subscriptions[0]

	// Update the local subscription based on the event type
	switch eventType {
	case "customer.subscription.created":
		// This should rarely happen as we create subscriptions ourselves, but just in case
		subscriptionModel.Status = enum.SubscriptionStatus(stripeSubscription.Status)
		subscriptionModel.CurrentPeriodStart = time.Unix(stripeSubscription.CurrentPeriodStart, 0)
		subscriptionModel.CurrentPeriodEnd = time.Unix(stripeSubscription.CurrentPeriodEnd, 0)
		subscriptionModel.CancelAtPeriodEnd = stripeSubscription.CancelAtPeriodEnd

	case "customer.subscription.updated":
		subscriptionModel.Status = enum.SubscriptionStatus(stripeSubscription.Status)
		subscriptionModel.CurrentPeriodStart = time.Unix(stripeSubscription.CurrentPeriodStart, 0)
		subscriptionModel.CurrentPeriodEnd = time.Unix(stripeSubscription.CurrentPeriodEnd, 0)
		subscriptionModel.CancelAtPeriodEnd = stripeSubscription.CancelAtPeriodEnd
		if stripeSubscription.CanceledAt > 0 {
			canceledAt := time.Unix(stripeSubscription.CanceledAt, 0)
			subscriptionModel.CanceledAt = &canceledAt
		}

	case "customer.subscription.deleted":
		subscriptionModel.Status = enum.SubscriptionStatusCanceled
		canceledAt := time.Now()
		subscriptionModel.CanceledAt = &canceledAt
	}

	// Update the subscription in the database
	err = sp.subscriptionService.Update(ctx, subscriptionModel)
	if err != nil {
		return fmt.Errorf("failed to update local subscription: %w", err)
	}

	return nil
}

// handleInvoiceEvent handles invoice-related webhook events
func (sp *StripePayment) handleInvoiceEvent(ctx context.Context, stripeInvoice *stripe.Invoice, eventType stripe.EventType) error {
	// Find the local invoice
	invoices, err := sp.invoiceService.ListByStripeID(ctx, stripeInvoice.ID)
	if err != nil || len(invoices) == 0 {
		return fmt.Errorf("failed to find local invoice for Stripe ID %s: %w", stripeInvoice.ID, err)
	}
	invoiceModel := invoices[0]

	// Update the local invoice based on the event type
	switch eventType {
	case "invoice.paid":
		invoiceModel.Status = enum.InvoiceStatusPaid
		invoiceModel.AmountPaid = uint64(stripeInvoice.AmountPaid)
		invoiceModel.AmountRemaining = uint64(stripeInvoice.AmountRemaining)
		invoiceModel.PaidAt = time.Now()

	case "invoice.payment_failed":
		invoiceModel.Status = enum.InvoiceStatusPaymentFailed
		invoiceModel.AmountPaid = uint64(stripeInvoice.AmountPaid)
		invoiceModel.AmountRemaining = uint64(stripeInvoice.AmountRemaining)
	}

	// Update the invoice in the database
	err = sp.invoiceService.Update(ctx, invoiceModel)
	if err != nil {
		return fmt.Errorf("failed to update local invoice: %w", err)
	}

	return nil
}
