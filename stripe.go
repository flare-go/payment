package payment

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/client"
	"github.com/stripe/stripe-go/v79/webhook"

	"goflare.io/payment/config"
	"goflare.io/payment/customer"
	"goflare.io/payment/invoice"
	"goflare.io/payment/models"
	"goflare.io/payment/models/enum"
	"goflare.io/payment/payment_intent"
	"goflare.io/payment/payment_method"
	"goflare.io/payment/price"
	"goflare.io/payment/product"
	"goflare.io/payment/refund"
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
	refundService        refund.Service
}

func NewStripePayment(config *config.Config,
	cs customer.Service,
	ps product.Service,
	prs price.Service,
	ss subscription.Service,
	is invoice.Service,
	pms payment_method.Service,
	pis payment_intent.Service,
	rs refund.Service) Payment {
	return &StripePayment{
		client:               client.New(config.Stripe.SecretKey, nil),
		customerService:      cs,
		productService:       ps,
		priceService:         prs,
		subscriptionService:  ss,
		invoiceService:       is,
		paymentMethodService: pms,
		paymentIntentService: pis,
		refundService:        rs,
	}
}

// CreateCustomer creates a new customer in Stripe and in the local database
func (sp *StripePayment) CreateCustomer(ctx context.Context, userID uint64, email, name string) error {
	params := &stripe.CustomerParams{
		Email: stripe.String(email),
		Name:  stripe.String(name),
		Metadata: map[string]string{
			"user_id": strconv.FormatUint(userID, 10),
		},
	}
	stripeCustomer, err := sp.client.Customers.New(params)
	if err != nil {
		return fmt.Errorf("failed to create Stripe customer: %w", err)
	}

	customerModel := &models.Customer{
		UserID:   userID,
		Name:     name,
		Email:    email,
		StripeID: stripeCustomer.ID,
	}
	if err = sp.customerService.Create(ctx, customerModel); err != nil {
		return fmt.Errorf("failed to create local customer record: %w", err)
	}

	return nil
}

// GetCustomer retrieves a customer from the local database
func (sp *StripePayment) GetCustomer(ctx context.Context, customerID uint64) (*models.Customer, error) {
	return sp.customerService.GetByID(ctx, customerID)
}

// UpdateCustomerBalance updates a customer in Stripe and in the local database
func (sp *StripePayment) UpdateCustomerBalance(ctx context.Context, updateCustomer *models.Customer) error {

	params := &stripe.CustomerParams{
		Balance: &updateCustomer.Balance,
	}

	if _, err := sp.client.Customers.Update(updateCustomer.StripeID, params); err != nil {
		return fmt.Errorf("failed to update Stripe customer: %w", err)
	}

	if err := sp.customerService.UpdateBalance(ctx, updateCustomer.ID, uint64(updateCustomer.Balance)); err != nil {
		return fmt.Errorf("failed to update local customer record: %w", err)
	}

	return nil
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
func (sp *StripePayment) CreateProduct(ctx context.Context, req models.Product) error {
	productParams := &stripe.ProductParams{
		Name:        stripe.String(req.Name),
		Description: stripe.String(req.Description),
		Active:      stripe.Bool(req.Active),
		Metadata:    req.Metadata,
	}
	stripeProduct, err := sp.client.Products.New(productParams)
	if err != nil {
		return fmt.Errorf("failed to create Stripe product: %w", err)
	}

	// 創建本地 Product
	productModel := &models.Product{
		Name:        req.Name,
		Description: req.Description,
		Active:      req.Active,
		Metadata:    req.Metadata,
		StripeID:    stripeProduct.ID,
	}

	if err = sp.productService.Create(ctx, productModel); err != nil {
		return fmt.Errorf("failed to create local product: %w", err)
	}

	// 創建 Prices
	var prices []*models.Price
	for _, priceReq := range req.Prices {
		priceParams := &stripe.PriceParams{
			Product:    stripe.String(stripeProduct.ID),
			Currency:   stripe.String(string(priceReq.Currency)),
			UnitAmount: stripe.Int64(int64(priceReq.UnitAmount * 100)), // 轉換為最小單位
		}

		if priceReq.Type == enum.PriceTypeRecurring {
			priceParams.Recurring = &stripe.PriceRecurringParams{
				Interval:      stripe.String(string(priceReq.RecurringInterval)),
				IntervalCount: stripe.Int64(int64(priceReq.RecurringIntervalCount)),
			}
			if priceReq.TrialPeriodDays != 0 {
				priceParams.Recurring.TrialPeriodDays = stripe.Int64(int64(priceReq.TrialPeriodDays))
			}
		}

		stripePrice, err := sp.client.Prices.New(priceParams)
		if err != nil {
			return fmt.Errorf("failed to create Stripe price: %w", err)
		}

		priceModel := &models.Price{
			ProductID:              productModel.ID,
			Type:                   priceReq.Type,
			Currency:               priceReq.Currency,
			UnitAmount:             priceReq.UnitAmount,
			RecurringInterval:      priceReq.RecurringInterval,
			RecurringIntervalCount: priceReq.RecurringIntervalCount,
			TrialPeriodDays:        priceReq.TrialPeriodDays,
			StripeID:               stripePrice.ID,
		}

		if err = sp.priceService.Create(ctx, priceModel); err != nil {
			return fmt.Errorf("failed to create local price: %w", err)
		}

		prices = append(prices, priceModel)
	}

	return nil
}

func (sp *StripePayment) GetProductWithActivePrices(ctx context.Context, productID uint64) (*models.Product, error) {
	productModel, err := sp.productService.GetByID(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("failed to get Stripe product: %w", err)
	}

	prices, err := sp.priceService.ListActive(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("failed to list Stripe prices: %w", err)
	}

	productModel.Prices = prices
	return productModel, nil
}

// GetProductWithAllPrices retrieves a product from the local database
func (sp *StripePayment) GetProductWithAllPrices(ctx context.Context, productID uint64) (*models.Product, error) {
	productModel, err := sp.productService.GetByID(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("failed to get Stripe product: %w", err)
	}

	prices, err := sp.priceService.List(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("failed to list Stripe prices: %w", err)
	}

	productModel.Prices = prices
	return productModel, nil
}

// UpdateProduct updates a product in Stripe and in the local database
func (sp *StripePayment) UpdateProduct(ctx context.Context, product *models.Product) error {
	params := &stripe.ProductParams{
		Name:        stripe.String(product.Name),
		Description: stripe.String(product.Description),
		Active:      stripe.Bool(product.Active),
		Metadata:    product.Metadata,
	}

	if _, err := sp.client.Products.Update(product.StripeID, params); err != nil {
		return fmt.Errorf("failed to update Stripe product: %w", err)
	}

	if err := sp.productService.Update(ctx, product); err != nil {
		return fmt.Errorf("failed to update local product record: %w", err)
	}

	return nil
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
func (sp *StripePayment) ListProducts(ctx context.Context) ([]*models.Product, error) {
	return sp.productService.List(ctx, 1000, 0) // Assuming a large limit, you might want to implement pagination
}

// CreatePrice creates a new price in Stripe and in the local database
func (sp *StripePayment) CreatePrice(ctx context.Context, price models.Price) error {
	productModel, err := sp.productService.GetByID(ctx, price.ProductID)
	if err != nil {
		return fmt.Errorf("failed to get product: %w", err)
	}

	params := &stripe.PriceParams{
		Product:    stripe.String(productModel.StripeID),
		Currency:   stripe.String(string(price.Currency)),
		UnitAmount: stripe.Int64(int64(price.UnitAmount * 100)),
	}

	if price.Type == enum.PriceTypeRecurring {
		params.Recurring = &stripe.PriceRecurringParams{
			Interval:        stripe.String(string(price.RecurringInterval)),
			IntervalCount:   stripe.Int64(int64(price.RecurringIntervalCount)),
			TrialPeriodDays: stripe.Int64(int64(price.TrialPeriodDays)),
		}
	}

	stripePrice, err := sp.client.Prices.New(params)
	if err != nil {
		return fmt.Errorf("failed to create Stripe price: %w", err)
	}

	price.StripeID = stripePrice.ID
	if err = sp.priceService.Create(ctx, &price); err != nil {
		return fmt.Errorf("failed to create local price record: %w", err)
	}

	return nil
}

// DeletePrice deletes a price from Stripe and from the local database
func (sp *StripePayment) DeletePrice(ctx context.Context, priceID uint64) error {
	priceModel, err := sp.priceService.GetByID(ctx, priceID)
	if err != nil {
		return fmt.Errorf("failed to get price: %w", err)
	}

	// In Stripe, you can't delete prices, you can only deactivate them
	if _, err = sp.client.Prices.Update(priceModel.StripeID, &stripe.PriceParams{
		Active: stripe.Bool(false),
	}); err != nil {
		return fmt.Errorf("failed to deactivate Stripe price: %w", err)
	}

	if err = sp.priceService.Delete(ctx, priceID); err != nil {
		return fmt.Errorf("failed to delete local price record: %w", err)
	}

	return nil
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

	if err = sp.subscriptionService.Create(ctx, subscriptionModel); err != nil {
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

	if err = sp.subscriptionService.Update(ctx, subscription); err != nil {
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

	if err = sp.subscriptionService.Update(ctx, subscriptionModel); err != nil {
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
		AmountDue:       float64(stripeInvoice.AmountDue),
		AmountPaid:      float64(stripeInvoice.AmountPaid),
		AmountRemaining: float64(stripeInvoice.AmountRemaining),
		DueDate:         time.Unix(stripeInvoice.DueDate, 0),
		StripeID:        stripeInvoice.ID,
	}

	if err = sp.invoiceService.Create(ctx, invoiceModel); err != nil {
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
	invoiceModel.AmountPaid = float64(stripeInvoice.AmountPaid)
	invoiceModel.AmountRemaining = float64(stripeInvoice.AmountRemaining)
	if stripeInvoice.Status == stripe.InvoiceStatusPaid {
		invoiceModel.PaidAt = time.Now()
	}

	if err = sp.invoiceService.Update(ctx, invoiceModel); err != nil {
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
	if _, err = sp.client.PaymentMethods.Attach(stripePaymentMethod.ID, &stripe.PaymentMethodAttachParams{
		Customer: stripe.String(customerModel.StripeID),
	}); err != nil {
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
	if err := sp.paymentMethodService.Update(ctx, paymentMethod); err != nil {
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

	if _, err = sp.client.PaymentMethods.Detach(paymentMethod.StripeID, nil); err != nil {
		return fmt.Errorf("failed to detach Stripe payment method: %w", err)
	}

	if err = sp.paymentMethodService.Delete(ctx, paymentMethodID); err != nil {
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
		Amount:       float64(amount),
		Currency:     currency,
		Status:       enum.PaymentIntentStatus(stripePaymentIntent.Status),
		ClientSecret: stripePaymentIntent.ClientSecret,
		StripeID:     stripePaymentIntent.ID,
	}

	if err = sp.paymentIntentService.Create(ctx, paymentIntent); err != nil {
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

	if err = sp.paymentIntentService.Update(ctx, paymentIntent); err != nil {
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

	if err = sp.paymentIntentService.Update(ctx, paymentIntent); err != nil {
		return nil, fmt.Errorf("failed to update local payment intent record: %w", err)
	}

	return paymentIntent, nil
}

func (sp *StripePayment) CreateRefund(ctx context.Context, paymentIntentID uint64, amount float64, reason string) (*models.Refund, error) {
	params := &stripe.RefundParams{
		PaymentIntent: stripe.String(strconv.FormatUint(paymentIntentID, 10)),
		Amount:        stripe.Int64(int64(amount * 100)), // Convert to cents
		Reason:        stripe.String(reason),
	}

	stripeRefund, err := sp.client.Refunds.New(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create Stripe refund: %w", err)
	}

	refundModel := &models.Refund{
		PaymentIntentID: paymentIntentID,
		Amount:          float64(stripeRefund.Amount) / 100, // Convert back to dollars
		Status:          enum.RefundStatus(stripeRefund.Status),
		Reason:          reason,
		StripeID:        stripeRefund.ID,
	}

	if err = sp.refundService.Create(ctx, refundModel); err != nil {
		return nil, fmt.Errorf("failed to create local refund record: %w", err)
	}

	return refundModel, nil
}

// GetRefund retrieves a refund from the local database
func (sp *StripePayment) GetRefund(ctx context.Context, refundID uint64) (*models.Refund, error) {
	return sp.refundService.GetByID(ctx, refundID)
}

// UpdateRefund updates a refund in Stripe and in the local database
func (sp *StripePayment) UpdateRefund(ctx context.Context, refundID uint64, reason string) (*models.Refund, error) {
	refundModel, err := sp.refundService.GetByID(ctx, refundID)
	if err != nil {
		return nil, fmt.Errorf("failed to get refund: %w", err)
	}

	params := &stripe.RefundParams{
		Reason: stripe.String(reason),
	}

	stripeRefund, err := sp.client.Refunds.Update(refundModel.StripeID, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update Stripe refund: %w", err)
	}

	refundModel.Reason = reason
	refundModel.Status = enum.RefundStatus(stripeRefund.Status)

	if err = sp.refundService.UpdateStatus(ctx, refundID, refundModel.Status, refundModel.Reason); err != nil {
		return nil, fmt.Errorf("failed to update local refund record: %w", err)
	}

	return refundModel, nil
}

// ListRefunds lists all refunds for a payment intent from the local database
func (sp *StripePayment) ListRefunds(ctx context.Context, paymentIntentID uint64) ([]*models.Refund, error) {
	return sp.refundService.ListByPaymentIntentID(ctx, paymentIntentID)
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
		if err = json.Unmarshal(event.Data.Raw, &subscriptionStripeModel); err != nil {
			return fmt.Errorf("failed to unmarshal subscription data: %w", err)
		}
		return sp.handleSubscriptionEvent(ctx, &subscriptionStripeModel, event.Type)

	case "invoice.paid", "invoice.payment_failed":
		var invoiceStripeModel stripe.Invoice
		if err = json.Unmarshal(event.Data.Raw, &invoiceStripeModel); err != nil {
			return fmt.Errorf("failed to unmarshal invoice data: %w", err)
		}
		return sp.handleInvoiceEvent(ctx, &invoiceStripeModel, event.Type)
	case "charge.refunded":
		var chargeStripeModel stripe.Charge
		if err = json.Unmarshal(event.Data.Raw, &chargeStripeModel); err != nil {
			return fmt.Errorf("failed to unmarshal charge data: %w", err)
		}
		return sp.handleRefundEvent(ctx, &chargeStripeModel)

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
	if err = sp.subscriptionService.Update(ctx, subscriptionModel); err != nil {
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
		invoiceModel.AmountPaid = float64(stripeInvoice.AmountPaid)
		invoiceModel.AmountRemaining = float64(stripeInvoice.AmountRemaining)
		invoiceModel.PaidAt = time.Now()

	case "invoice.payment_failed":
		invoiceModel.Status = enum.InvoiceStatusPaymentFailed
		invoiceModel.AmountPaid = float64(stripeInvoice.AmountPaid)
		invoiceModel.AmountRemaining = float64(stripeInvoice.AmountRemaining)
	}

	// Update the invoice in the database
	if err = sp.invoiceService.Update(ctx, invoiceModel); err != nil {
		return fmt.Errorf("failed to update local invoice: %w", err)
	}

	return nil
}

func (sp *StripePayment) handleRefundEvent(ctx context.Context, stripeCharge *stripe.Charge) error {
	for _, stripeRefund := range stripeCharge.Refunds.Data {
		refunds, err := sp.refundService.ListByStripeID(ctx, stripeRefund.ID)
		if err != nil || len(refunds) == 0 {
			// 如果本地數據庫中沒有找到對應的退款記錄，創建一個新的
			refundModel := &models.Refund{
				Amount:   float64(stripeRefund.Amount) / 100,
				Status:   enum.RefundStatus(stripeRefund.Status),
				Reason:   string(stripeRefund.Reason),
				StripeID: stripeRefund.ID,
			}
			if err = sp.refundService.Create(ctx, refundModel); err != nil {
				return fmt.Errorf("failed to create local refund record: %w", err)
			}
		} else {
			// 如果找到了對應的退款記錄，更新它
			refundModel := refunds[0]
			refundModel.Status = enum.RefundStatus(stripeRefund.Status)
			refundModel.Reason = string(stripeRefund.Reason)
			if err = sp.refundService.UpdateStatus(ctx, refundModel.ID, refundModel.Status, refundModel.Reason); err != nil {
				return fmt.Errorf("failed to update local refund record: %w", err)
			}
		}
	}
	return nil
}
