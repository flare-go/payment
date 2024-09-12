package payment

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/nats-io/nats.go"
	"strings"
	"time"

	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/client"
	"github.com/stripe/stripe-go/v79/webhook"
	"go.uber.org/zap"

	"goflare.io/payment/charge"
	"goflare.io/payment/checkout_session"
	"goflare.io/payment/config"
	"goflare.io/payment/coupon"
	"goflare.io/payment/customer"
	"goflare.io/payment/discount"
	"goflare.io/payment/disputes"
	"goflare.io/payment/event"
	"goflare.io/payment/invoice"
	"goflare.io/payment/models"
	"goflare.io/payment/payment_intent"
	"goflare.io/payment/payment_link"
	"goflare.io/payment/payment_method"
	"goflare.io/payment/price"
	"goflare.io/payment/product"
	"goflare.io/payment/promotion_code"
	"goflare.io/payment/quote"
	"goflare.io/payment/refund"
	"goflare.io/payment/review"
	"goflare.io/payment/subscription"
	"goflare.io/payment/tax_rate"
)

type StripePayment struct {
	client        *client.API
	natsConn      *nats.Conn
	subscriptions []*nats.Subscription

	charge          charge.Service
	checkoutSession checkout_session.Service
	coupon          coupon.Service
	customer        customer.Service
	discount        discount.Service
	dispute         disputes.Service
	event           event.Service
	invoice         invoice.Service
	paymentIntent   payment_intent.Service
	paymentLink     payment_link.Service
	paymentMethod   payment_method.Service
	price           price.Service
	product         product.Service
	promotionCode   promotion_code.Service
	quote           quote.Service
	refund          refund.Service
	review          review.Service
	subscription    subscription.Service
	taxRate         tax_rate.Service

	logger *zap.Logger
}

func NewStripePayment(config *config.Config,
	cs customer.Service,
	charge charge.Service,
	coupon coupon.Service,
	checkoutSession checkout_session.Service,
	discount discount.Service,
	dispute disputes.Service,
	event event.Service,
	ps product.Service,
	prs price.Service,
	ss subscription.Service,
	is invoice.Service,
	pms payment_method.Service,
	paymentLink payment_link.Service,
	pis payment_intent.Service,
	pcs promotion_code.Service,
	rs refund.Service,
	review review.Service,
	taxRate tax_rate.Service,
	quote quote.Service,
	logger *zap.Logger) Payment {
	sp := &StripePayment{
		client:          client.New(config.Stripe.SecretKey, nil),
		charge:          charge,
		coupon:          coupon,
		checkoutSession: checkoutSession,
		customer:        cs,
		discount:        discount,
		dispute:         dispute,
		event:           event,
		product:         ps,
		price:           prs,
		subscription:    ss,
		invoice:         is,
		paymentMethod:   pms,
		paymentLink:     paymentLink,
		promotionCode:   pcs,
		paymentIntent:   pis,
		quote:           quote,
		review:          review,
		taxRate:         taxRate,
		refund:          rs,
		logger:          logger,
	}
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		logger.Error("error connecting to nats", zap.Error(err))
	}

	sp.natsConn = nc

	sp.subscriptions = append(sp.subscriptions, sp.subscribeToEvent(sp.handleEvent, NatsSubjectAll))
	sp.subscriptions = append(sp.subscriptions, sp.subscribeToEvent(sp.handleProcessedEvent, NatsSubjectProcessed))
	sp.subscriptions = append(sp.subscriptions, sp.subscribeToEvent(sp.handleCustomerEvent, NatsSubjectCustomer, NatsSubjectCreated))
	sp.subscriptions = append(sp.subscriptions, sp.subscribeToEvent(sp.handleCustomerEvent, NatsSubjectCustomer, NatsSubjectUpdated))
	sp.subscriptions = append(sp.subscriptions, sp.subscribeToEvent(sp.handleCustomerEvent, NatsSubjectCustomer, NatsSubjectDeleted))
	sp.subscriptions = append(sp.subscriptions, sp.subscribeToEvent(sp.handleSubscriptionEvent, NatsSubjectCustomer, NatsSubjectSubscription, NatsSubjectSingleAll))
	sp.subscriptions = append(sp.subscriptions, sp.subscribeToEvent(sp.handleInvoiceEvent, NatsSubjectInvoice, NatsSubjectSingleAll))
	sp.subscriptions = append(sp.subscriptions, sp.subscribeToEvent(sp.handlePaymentIntentEvent, NatsSubjectPaymentIntent, NatsSubjectSingleAll))
	sp.subscriptions = append(sp.subscriptions, sp.subscribeToEvent(sp.handleChargeEvent, NatsSubjectCharge, NatsSubjectSucceeded))
	sp.subscriptions = append(sp.subscriptions, sp.subscribeToEvent(sp.handleChargeEvent, NatsSubjectCharge, NatsSubjectFailed))
	sp.subscriptions = append(sp.subscriptions, sp.subscribeToEvent(sp.handleChargeEvent, NatsSubjectCharge, NatsSubjectRefunded))
	sp.subscriptions = append(sp.subscriptions, sp.subscribeToEvent(sp.handleChargeEvent, NatsSubjectCharge, NatsSubjectCaptured))
	sp.subscriptions = append(sp.subscriptions, sp.subscribeToEvent(sp.handleChargeEvent, NatsSubjectCharge, NatsSubjectExpired))
	sp.subscriptions = append(sp.subscriptions, sp.subscribeToEvent(sp.handleChargeEvent, NatsSubjectCharge, NatsSubjectUpdated))
	sp.subscriptions = append(sp.subscriptions, sp.subscribeToEvent(sp.handleDisputeEvent, NatsSubjectCharge, NatsSubjectDispute, NatsSubjectSingleAll))
	sp.subscriptions = append(sp.subscriptions, sp.subscribeToEvent(sp.handleProductEvent, NatsSubjectProduct, NatsSubjectSingleAll))
	sp.subscriptions = append(sp.subscriptions, sp.subscribeToEvent(sp.handlePriceEvent, NatsSubjectPrice, NatsSubjectSingleAll))
	sp.subscriptions = append(sp.subscriptions, sp.subscribeToEvent(sp.handlePaymentMethodEvent, NatsSubjectPaymentMethod, NatsSubjectSingleAll))
	sp.subscriptions = append(sp.subscriptions, sp.subscribeToEvent(sp.handleCouponEvent, NatsSubjectCoupon, NatsSubjectSingleAll))
	sp.subscriptions = append(sp.subscriptions, sp.subscribeToEvent(sp.handleDiscountEvent, NatsSubjectCustomer, NatsSubjectDiscount, NatsSubjectSingleAll))
	sp.subscriptions = append(sp.subscriptions, sp.subscribeToEvent(sp.handleCheckoutSessionEvent, NatsSubjectCheckout, NatsSubjectSession, NatsSubjectSingleAll))
	sp.subscriptions = append(sp.subscriptions, sp.subscribeToEvent(sp.handleRefundEvent, NatsSubjectRefund, NatsSubjectSingleAll))
	sp.subscriptions = append(sp.subscriptions, sp.subscribeToEvent(sp.handlePromotionCodeEvent, NatsSubjectPromotionCode, NatsSubjectSingleAll))
	sp.subscriptions = append(sp.subscriptions, sp.subscribeToEvent(sp.handleQuoteEvent, NatsSubjectQuote, NatsSubjectSingleAll))
	sp.subscriptions = append(sp.subscriptions, sp.subscribeToEvent(sp.handlePaymentLinkEvent, NatsSubjectPaymentLink, NatsSubjectSingleAll))
	sp.subscriptions = append(sp.subscriptions, sp.subscribeToEvent(sp.handleTaxRateEvent, NatsSubjectTaxRate, NatsSubjectSingleAll))
	sp.subscriptions = append(sp.subscriptions, sp.subscribeToEvent(sp.handleReviewEvent, NatsSubjectReview, NatsSubjectSingleAll))

	return sp
}

func (sp *StripePayment) subscribeToEvent(handler func(msg *nats.Msg), subject ...string) *nats.Subscription {
	subj := "stripe.event" + strings.Join(subject, "")
	sub, err := sp.natsConn.Subscribe(subj, handler)
	if err != nil {
		sp.logger.Error("Failed to subscribe to event", zap.Error(err), zap.String("subject", subj))
		return nil
	}
	return sub
}

// CreateCustomer creates a new customer in Stripe and in the local database
func (sp *StripePayment) CreateCustomer(ctx context.Context, email, name string) error {
	params := &stripe.CustomerParams{
		Email: stripe.String(email),
		Name:  stripe.String(name),
	}
	stripeCustomer, err := sp.client.Customers.New(params)
	if err != nil {
		return fmt.Errorf("failed to create Stripe customer: %w", err)
	}

	customerModel := &models.Customer{
		ID:    stripeCustomer.ID,
		Name:  name,
		Email: email,
	}
	if err = sp.customer.Create(ctx, customerModel); err != nil {
		return fmt.Errorf("failed to create local customer record: %w", err)
	}

	return nil
}

// GetCustomer retrieves a customer from the local database
func (sp *StripePayment) GetCustomer(ctx context.Context, customerID string) (*models.Customer, error) {
	return sp.customer.GetByID(ctx, customerID)
}

// UpdateCustomerBalance updates a customer in Stripe and in the local database
func (sp *StripePayment) UpdateCustomerBalance(updateCustomer *models.Customer) error {

	params := &stripe.CustomerParams{
		Balance: &updateCustomer.Balance,
	}

	if _, err := sp.client.Customers.Update(updateCustomer.ID, params); err != nil {
		return fmt.Errorf("failed to update Stripe customer: %w", err)
	}

	return nil
}

// DeleteCustomer deletes a customer from Stripe and from the local database
func (sp *StripePayment) DeleteCustomer(customerID string) error {
	if _, err := sp.client.Customers.Del(customerID, nil); err != nil {
		return fmt.Errorf("failed to delete Stripe customer: %w", err)
	}
	return nil
}

// CreateProduct creates a new product in Stripe and in the local database
func (sp *StripePayment) CreateProduct(req models.Product) error {
	productParams := &stripe.ProductParams{
		Name:        stripe.String(req.Name),
		Description: stripe.String(req.Description),
		Active:      stripe.Bool(req.Active),
		Metadata:    req.Metadata,
	}
	if _, err := sp.client.Products.New(productParams); err != nil {
		return fmt.Errorf("failed to create Stripe product: %w", err)
	}

	return nil
}

func (sp *StripePayment) GetProductWithActivePrices(ctx context.Context, productID string) (*models.Product, error) {
	productModel, err := sp.product.GetByID(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("failed to get Stripe product: %w", err)
	}

	prices, err := sp.price.ListActive(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("failed to list Stripe prices: %w", err)
	}

	productModel.Prices = prices
	return productModel, nil
}

// GetProductWithAllPrices retrieves a product from the local database
func (sp *StripePayment) GetProductWithAllPrices(ctx context.Context, productID string) (*models.Product, error) {
	productModel, err := sp.product.GetByID(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("failed to get Stripe product: %w", err)
	}

	prices, err := sp.price.List(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("failed to list Stripe prices: %w", err)
	}

	productModel.Prices = prices
	return productModel, nil
}

// UpdateProduct updates a product in Stripe and in the local database
func (sp *StripePayment) UpdateProduct(product *models.Product) error {
	params := &stripe.ProductParams{
		Name:        stripe.String(product.Name),
		Description: stripe.String(product.Description),
		Active:      stripe.Bool(product.Active),
		Metadata:    product.Metadata,
	}

	if _, err := sp.client.Products.Update(product.ID, params); err != nil {
		return fmt.Errorf("failed to update Stripe product: %w", err)
	}

	return nil
}

// DeleteProduct deletes a product from Stripe and from the local database
func (sp *StripePayment) DeleteProduct(productID string) error {

	if _, err := sp.client.Products.Del(productID, nil); err != nil {
		return fmt.Errorf("failed to delete Stripe product: %w", err)
	}

	return nil
}

// ListProducts lists all products from the local database
func (sp *StripePayment) ListProducts(ctx context.Context) ([]*models.Product, error) {
	return sp.product.List(ctx, 1000, 0) // Assuming a large limit, you might want to implement pagination
}

// CreatePrice creates a new price in Stripe and in the local database
func (sp *StripePayment) CreatePrice(price models.Price) error {

	params := &stripe.PriceParams{
		Product:    stripe.String(price.ProductID),
		Currency:   stripe.String(string(price.Currency)),
		UnitAmount: stripe.Int64(int64(price.UnitAmount * 100)),
	}

	if price.Type == stripe.PriceTypeRecurring {
		params.Recurring = &stripe.PriceRecurringParams{
			Interval:        stripe.String(string(price.RecurringInterval)),
			IntervalCount:   stripe.Int64(int64(price.RecurringIntervalCount)),
			TrialPeriodDays: stripe.Int64(int64(price.TrialPeriodDays)),
		}
	}

	_, err := sp.client.Prices.New(params)
	if err != nil {
		return fmt.Errorf("failed to create Stripe price: %w", err)
	}

	return nil
}

// DeletePrice deletes a price from Stripe and from the local database
func (sp *StripePayment) DeletePrice(priceID string) error {
	// In Stripe, you can't delete prices, you can only deactivate them
	if _, err := sp.client.Prices.Update(priceID, &stripe.PriceParams{
		Active: stripe.Bool(false),
	}); err != nil {
		return fmt.Errorf("failed to deactivate Stripe price: %w", err)
	}

	return nil
}

// CreateSubscription creates a new subscription in Stripe and in the local database
func (sp *StripePayment) CreateSubscription(customerID, priceID string) error {

	params := &stripe.SubscriptionParams{
		Customer: stripe.String(customerID),
		Items: []*stripe.SubscriptionItemsParams{
			{
				Price: stripe.String(priceID),
			},
		},
	}

	if _, err := sp.client.Subscriptions.New(params); err != nil {
		return fmt.Errorf("failed to create Stripe subscription: %w", err)
	}

	return nil
}

// GetSubscription retrieves a subscription from the local database
func (sp *StripePayment) GetSubscription(ctx context.Context, subscriptionID string) (*models.Subscription, error) {
	return sp.subscription.GetByID(ctx, subscriptionID)
}

// UpdateSubscription updates a subscription in Stripe and in the local database
func (sp *StripePayment) UpdateSubscription(subscription *models.Subscription) error {
	params := &stripe.SubscriptionParams{
		CancelAtPeriodEnd: stripe.Bool(subscription.CancelAtPeriodEnd),
	}
	if _, err := sp.client.Subscriptions.Update(subscription.ID, params); err != nil {
		return fmt.Errorf("failed to update Stripe subscription: %w", err)
	}
	return nil
}

// CancelSubscription cancels a subscription in Stripe and updates the local database
func (sp *StripePayment) CancelSubscription(subscriptionID string, cancelAtPeriodEnd bool) error {

	params := &stripe.SubscriptionParams{
		CancelAtPeriodEnd: stripe.Bool(cancelAtPeriodEnd),
	}

	if _, err := sp.client.Subscriptions.Update(subscriptionID, params); err != nil {
		return fmt.Errorf("failed to cancel Stripe subscription: %w", err)
	}

	return nil
}

// ListSubscriptions lists all subscriptions for a customer from the local database
func (sp *StripePayment) ListSubscriptions(ctx context.Context, customerID string) ([]*models.Subscription, error) {
	return sp.subscription.List(ctx, customerID, 1000, 0)
	// Assuming a large limit, you might want to implement pagination
}

// CreateInvoice creates a new invoice in Stripe and in the local database
func (sp *StripePayment) CreateInvoice(customerID, subscriptionID string) error {

	params := &stripe.InvoiceParams{
		Customer:     stripe.String(customerID),
		Subscription: stripe.String(subscriptionID),
	}

	if _, err := sp.client.Invoices.New(params); err != nil {
		return fmt.Errorf("failed to create Stripe invoice: %w", err)
	}

	return nil
}

// GetInvoice retrieves an invoice from the local database
func (sp *StripePayment) GetInvoice(ctx context.Context, invoiceID string) (*models.Invoice, error) {
	return sp.invoice.GetByID(ctx, invoiceID)
}

// PayInvoice pays an invoice in Stripe and updates the local database
func (sp *StripePayment) PayInvoice(invoiceID string) error {

	if _, err := sp.client.Invoices.Pay(invoiceID, nil); err != nil {
		return fmt.Errorf("failed to pay Stripe invoice: %w", err)
	}

	return nil
}

// ListInvoices lists all invoices for a customer from the local database
func (sp *StripePayment) ListInvoices(ctx context.Context, customerID string) ([]*models.Invoice, error) {
	return sp.invoice.List(ctx, customerID, 1000, 0)
	// Assuming a large limit, you might want to implement pagination
}

// GetPaymentMethod retrieves a payment method from the local database
func (sp *StripePayment) GetPaymentMethod(ctx context.Context, paymentMethodID string) (*models.PaymentMethod, error) {
	return sp.paymentMethod.GetByID(ctx, paymentMethodID)
}

// DeletePaymentMethod deletes a payment method from Stripe and from the local database
func (sp *StripePayment) DeletePaymentMethod(ctx context.Context, paymentMethodID string) error {

	if _, err := sp.client.PaymentMethods.Detach(paymentMethodID, nil); err != nil {
		return fmt.Errorf("failed to detach Stripe payment method: %w", err)
	}

	if err := sp.paymentMethod.Delete(ctx, paymentMethodID); err != nil {
		return fmt.Errorf("failed to delete local payment method record: %w", err)
	}

	return nil
}

// ListPaymentMethods lists all payment methods for a customer from the local database
func (sp *StripePayment) ListPaymentMethods(ctx context.Context, customerID string) ([]*models.PaymentMethod, error) {
	return sp.paymentMethod.List(ctx, customerID, 1000, 0)
	// Assuming a large limit, you might want to implement pagination
}

// CreatePaymentIntent creates a new payment intent in Stripe and in the local database
func (sp *StripePayment) CreatePaymentIntent(customerID, paymentMethodID string, amount uint64, currency stripe.Currency) error {

	params := &stripe.PaymentIntentParams{
		Amount:        stripe.Int64(int64(amount)),
		Currency:      stripe.String(string(currency)),
		Customer:      stripe.String(customerID),
		PaymentMethod: stripe.String(paymentMethodID),
	}

	if _, err := sp.client.PaymentIntents.New(params); err != nil {
		return fmt.Errorf("failed to create Stripe payment intent: %w", err)
	}

	return nil
}

// GetPaymentIntent retrieves a payment intent from the local database
func (sp *StripePayment) GetPaymentIntent(ctx context.Context, paymentIntentID string) (*models.PaymentIntent, error) {
	return sp.paymentIntent.GetByID(ctx, paymentIntentID)
}

// ConfirmPaymentIntent confirms a payment intent in Stripe and updates the local database
func (sp *StripePayment) ConfirmPaymentIntent(paymentIntentID, paymentMethodID string) error {

	params := &stripe.PaymentIntentConfirmParams{
		PaymentMethod: stripe.String(paymentMethodID),
	}

	if _, err := sp.client.PaymentIntents.Confirm(paymentIntentID, params); err != nil {
		return fmt.Errorf("failed to confirm Stripe payment intent: %w", err)
	}

	return nil
}

// CancelPaymentIntent cancels a payment intent in Stripe and updates the local database
func (sp *StripePayment) CancelPaymentIntent(paymentIntentID string) error {

	if _, err := sp.client.PaymentIntents.Cancel(paymentIntentID, nil); err != nil {
		return fmt.Errorf("failed to cancel Stripe payment intent: %w", err)
	}

	return nil
}

func (sp *StripePayment) ListPaymentIntent(ctx context.Context, limit, offset uint64) ([]*models.PaymentIntent, error) {
	return sp.paymentIntent.List(ctx, limit, offset)
}

func (sp *StripePayment) ListPaymentIntentByCustomerID(ctx context.Context, customerID string, limit, offset uint64) ([]*models.PaymentIntent, error) {
	return sp.paymentIntent.ListByCustomer(ctx, customerID, limit, offset)
}

func (sp *StripePayment) CreateRefund(paymentIntentID, reason string, amount uint64) error {
	params := &stripe.RefundParams{
		PaymentIntent: stripe.String(paymentIntentID),
		Amount:        stripe.Int64(int64(amount * 100)), // Convert to cents
		Reason:        stripe.String(reason),
	}

	if _, err := sp.client.Refunds.New(params); err != nil {
		return fmt.Errorf("failed to create Stripe refund: %w", err)
	}

	return nil
}

// GetRefund retrieves a refund from the local database
func (sp *StripePayment) GetRefund(ctx context.Context, refundID string) (*models.Refund, error) {
	return sp.refund.GetByID(ctx, refundID)
}

// UpdateRefund updates a refund in Stripe and in the local database
func (sp *StripePayment) UpdateRefund(refundID, reason string) error {

	params := &stripe.RefundParams{
		Reason: stripe.String(reason),
	}

	if _, err := sp.client.Refunds.Update(refundID, params); err != nil {
		return fmt.Errorf("failed to update Stripe refund: %w", err)
	}

	return nil
}

// ListRefunds lists all refunds for a payment intent from the local database
func (sp *StripePayment) ListRefunds(ctx context.Context, chargeID string) ([]*models.Refund, error) {
	return sp.refund.ListByChargeID(ctx, chargeID)
}

// HandleStripeWebhook handles Stripe webhook events
func (sp *StripePayment) HandleStripeWebhook(ctx context.Context, payload []byte, signature string) error {
	stripeEvent, err := webhook.ConstructEvent(payload, signature, "秘密")
	if err != nil {
		return fmt.Errorf("failed to verify webhook signature: %w", err)
	}

	processed, err := sp.event.IsEventProcessed(ctx, stripeEvent.ID)
	if processed {
		sp.logger.Info("Event is already processed", zap.String("event_id", stripeEvent.ID))
		return nil
	}

	// 使用 CreateWorkRequest 方法創建工作請求

	subject := fmt.Sprintf("stripe.event.%s", stripeEvent.Type)
	if err = sp.natsConn.Publish(subject, payload); err != nil {
		return fmt.Errorf("failed to publish event to NATS: %w", err)
	}

	sp.logger.Info("Stripe event published to NATS",
		zap.String("event_id", stripeEvent.ID),
		zap.String("event_type", string(stripeEvent.Type)))

	return nil
}

func (sp *StripePayment) handleEvent(msg *nats.Msg) {

	ctx := context.Background()
	var stripeEvent stripe.Event
	if err := json.Unmarshal(msg.Data, &stripeEvent); err != nil {
		sp.logger.Error("Failed to unmarshal event", zap.Error(err))
		return
	}

	if msg.Subject == NatsSubjectEventProcessed {
		return
	}

	eventModel := &models.Event{
		ID:        stripeEvent.ID,
		Type:      stripeEvent.Type,
		Processed: false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := sp.event.Create(ctx, eventModel); err != nil {
		sp.logger.Error("Failed to create event", zap.Error(err))
	}

	sp.logger.Info("Stripe event created", zap.String("event_id", stripeEvent.ID))
}

func (sp *StripePayment) handleProcessedEvent(msg *nats.Msg) {

	var eventModel stripe.Event
	if err := json.Unmarshal(msg.Data, &eventModel); err != nil {
		sp.logger.Error("Failed to unmarshal customer event", zap.Error(err))
		return
	}

	if err := sp.event.MarkEventAsProcessed(context.Background(), eventModel.ID); err != nil {
		sp.logger.Error("Failed to mark event as processed", zap.Error(err))
	}

	sp.logger.Info("Stripe event processed", zap.String("event_id", msg.Subject))
}

func (sp *StripePayment) handleCustomerEvent(msg *nats.Msg) {

	sp.logger.Info("Stripe customer event received", zap.String("event_id", msg.Subject))

	ctx := context.Background()
	var eventModel stripe.Event
	if err := json.Unmarshal(msg.Data, &eventModel); err != nil {
		sp.logger.Error("Failed to unmarshal customer event", zap.Error(err))
		return
	}

	customerModel := new(stripe.Customer)
	if err := json.Unmarshal(eventModel.Data.Raw, &customerModel); err != nil {
		sp.logger.Error("Failed to unmarshal customer event", zap.Error(err))
		return
	}

	partialCustomer := &models.PartialCustomer{
		ID: customerModel.ID,
	}

	if customerModel.Email != "" {
		partialCustomer.Email = &customerModel.Email
	}

	if customerModel.Balance != 0 {
		balance := customerModel.Balance
		partialCustomer.Balance = &balance
	}
	if customerModel.Created > 0 {
		createdAt := time.Unix(customerModel.Created, 0)
		partialCustomer.CreatedAt = &createdAt
	}

	var err error
	switch eventModel.Type {
	case "customer.created", "customer.updated":
		err = sp.customer.Upsert(ctx, partialCustomer)
	case "customer.deleted":
		err = sp.customer.Delete(ctx, customerModel.ID)
	default:
		sp.logger.Error(fmt.Sprintf("unexpected customer event type: %s", eventModel.Type))
	}
	if err != nil {
		sp.logger.Error("Failed to upsert customer event", zap.Error(err))
	}

	if err = sp.natsConn.Publish(NatsSubjectEventProcessed, msg.Data); err != nil {
		sp.logger.Error("Failed to publish event to NATS", zap.Error(err))
	}

	sp.logger.Info("Stripe customer event processed", zap.String("event_id", msg.Subject))
}

func (sp *StripePayment) handleSubscriptionEvent(msg *nats.Msg) {

	sp.logger.Info("Stripe subscription event received", zap.String("event_id", msg.Subject))

	ctx := context.Background()
	var eventModel stripe.Event
	if err := json.Unmarshal(msg.Data, &eventModel); err != nil {
		sp.logger.Error("Failed to unmarshal subscription event", zap.Error(err))
		return
	}

	subscriptionModel := new(stripe.Subscription)
	if err := json.Unmarshal(eventModel.Data.Raw, &subscriptionModel); err != nil {
		sp.logger.Error("Failed to unmarshal subscription event", zap.Error(err))
		return
	}

	partialSubscription := &models.PartialSubscription{
		ID: subscriptionModel.ID,
	}

	if subscriptionModel.Customer != nil {
		partialSubscription.CustomerID = &subscriptionModel.Customer.ID
	}
	if subscriptionModel.Status != "" {
		partialSubscription.Status = &subscriptionModel.Status
	}
	if subscriptionModel.CurrentPeriodStart > 0 {
		start := time.Unix(subscriptionModel.CurrentPeriodStart, 0)
		partialSubscription.CurrentPeriodStart = &start
	}
	if subscriptionModel.CurrentPeriodEnd > 0 {
		end := time.Unix(subscriptionModel.CurrentPeriodEnd, 0)
		partialSubscription.CurrentPeriodEnd = &end
	}
	partialSubscription.CancelAtPeriodEnd = &subscriptionModel.CancelAtPeriodEnd
	if subscriptionModel.CanceledAt > 0 {
		canceledAt := time.Unix(subscriptionModel.CanceledAt, 0)
		partialSubscription.CanceledAt = &canceledAt
	}

	var err error
	switch eventModel.Type {
	case "customer.subscription.created", "customer.subscription.updated",
		"customer.subscription.trial_will_end", "customer.subscription.pending_update_applied",
		"customer.subscription.pending_update_expired", "customer.subscription.paused",
		"customer.subscription.resumed":
		err = sp.subscription.Upsert(ctx, partialSubscription)
	case "customer.subscription.deleted":
		err = sp.subscription.Delete(ctx, subscriptionModel.ID)
	default:
		sp.logger.Error(fmt.Sprintf("unexpected customer subscription event type: %s", eventModel.Type))
	}

	if err != nil {
		sp.logger.Error("Failed to upsert subscription event", zap.Error(err))
	}

	if err = sp.natsConn.Publish(NatsSubjectEventProcessed, msg.Data); err != nil {
		sp.logger.Error("Failed to publish event to NATS", zap.Error(err))
	}

	sp.logger.Info("Stripe subscription event processed", zap.String("event_id", msg.Subject))
}

func (sp *StripePayment) handleInvoiceEvent(msg *nats.Msg) {

	sp.logger.Info("Stripe invoice event received", zap.String("event_id", msg.Subject))

	ctx := context.Background()
	var eventModel stripe.Event
	if err := json.Unmarshal(msg.Data, &eventModel); err != nil {
		sp.logger.Error("Failed to unmarshal invoice event", zap.Error(err))
		return
	}

	invoiceModel := new(stripe.Invoice)
	if err := json.Unmarshal(eventModel.Data.Raw, &invoiceModel); err != nil {
		sp.logger.Error("Failed to unmarshal invoice event", zap.Error(err))
		return
	}

	partialInvoice := &models.PartialInvoice{
		ID: invoiceModel.ID,
	}

	if invoiceModel.Customer != nil {
		partialInvoice.CustomerID = &invoiceModel.Customer.ID
	}
	if invoiceModel.Subscription != nil {
		partialInvoice.SubscriptionID = &invoiceModel.Subscription.ID
	}
	if invoiceModel.Status != "" {
		partialInvoice.Status = &invoiceModel.Status
	}
	if invoiceModel.Currency != "" {
		partialInvoice.Currency = &invoiceModel.Currency
	}

	amountDue := float64(invoiceModel.AmountDue) / 100
	partialInvoice.AmountDue = &amountDue

	amountPaid := float64(invoiceModel.AmountPaid) / 100
	partialInvoice.AmountPaid = &amountPaid

	amountRemaining := float64(invoiceModel.AmountRemaining) / 100
	partialInvoice.AmountRemaining = &amountRemaining

	if invoiceModel.DueDate > 0 {
		dueDate := time.Unix(invoiceModel.DueDate, 0)
		partialInvoice.DueDate = &dueDate
	}
	if invoiceModel.Created > 0 {
		createdAt := time.Unix(invoiceModel.Created, 0)
		partialInvoice.CreatedAt = &createdAt
	}

	if invoiceModel.Status == stripe.InvoiceStatusPaid {
		now := time.Now()
		partialInvoice.PaidAt = &now
	}

	var err error
	switch eventModel.Type {
	case "invoice.created", "invoice.updated", "invoice.finalized",
		"invoice.payment_succeeded", "invoice.payment_failed", "invoice.sent", "invoice.paid":
		err = sp.invoice.Upsert(ctx, partialInvoice)
	case "invoice.deleted":
		err = sp.invoice.Delete(ctx, invoiceModel.ID)
	default:
		sp.logger.Error(fmt.Sprintf("unexpected invoice event type: %s", eventModel.Type))
	}
	if err != nil {
		sp.logger.Error("Failed to upsert invoice event", zap.Error(err))
	}

	if err = sp.natsConn.Publish(NatsSubjectEventProcessed, msg.Data); err != nil {
		sp.logger.Error("Failed to publish event to NATS", zap.Error(err))
	}

	sp.logger.Info("Stripe invoice event processed", zap.String("event_id", msg.Subject))
}

func (sp *StripePayment) handlePaymentIntentEvent(msg *nats.Msg) {

	sp.logger.Info("Stripe payment intent event received", zap.String("event_id", msg.Subject))

	ctx := context.Background()
	var eventModel stripe.Event
	if err := json.Unmarshal(msg.Data, &eventModel); err != nil {
		sp.logger.Error("Failed to unmarshal payment intent event", zap.Error(err))
		return
	}

	paymentIntent := new(stripe.PaymentIntent)
	if err := json.Unmarshal(eventModel.Data.Raw, &paymentIntent); err != nil {
		sp.logger.Error("Failed to unmarshal payment intent event", zap.Error(err))
		return
	}
	partialPaymentIntent := &models.PartialPaymentIntent{
		ID: paymentIntent.ID,
	}

	if paymentIntent.Customer != nil {
		partialPaymentIntent.CustomerID = &paymentIntent.Customer.ID
	}
	if paymentIntent.Amount > 0 {
		amount := float64(paymentIntent.Amount) / 100
		partialPaymentIntent.Amount = &amount
	}
	if paymentIntent.Currency != "" {
		partialPaymentIntent.Currency = &paymentIntent.Currency
	}
	if paymentIntent.Status != "" {
		partialPaymentIntent.Status = &paymentIntent.Status
	}
	if paymentIntent.PaymentMethod != nil {
		partialPaymentIntent.PaymentMethodID = &paymentIntent.PaymentMethod.ID
	}
	if paymentIntent.SetupFutureUsage != "" {
		partialPaymentIntent.SetupFutureUsage = &paymentIntent.SetupFutureUsage
	}
	if paymentIntent.ClientSecret != "" {
		partialPaymentIntent.ClientSecret = &paymentIntent.ClientSecret
	}
	if paymentIntent.CaptureMethod != "" {
		partialPaymentIntent.CaptureMethod = &paymentIntent.CaptureMethod
	}
	if paymentIntent.Created > 0 {
		createdAt := time.Unix(paymentIntent.Created, 0)
		partialPaymentIntent.CreatedAt = &createdAt
	}

	if err := sp.paymentIntent.Upsert(ctx, partialPaymentIntent); err != nil {
		sp.logger.Error("Failed to upsert payment intent", zap.Error(err))
	}

	if err := sp.natsConn.Publish(NatsSubjectEventProcessed, msg.Data); err != nil {
		sp.logger.Error("Failed to publish event to NATS", zap.Error(err))
	}

	sp.logger.Info("Stripe payment intent event processed", zap.String("event_id", msg.Subject))
}

func (sp *StripePayment) handleChargeEvent(msg *nats.Msg) {

	sp.logger.Info("Stripe charge event received", zap.String("event_id", msg.Subject))

	ctx := context.Background()
	var eventModel stripe.Event
	if err := json.Unmarshal(msg.Data, &eventModel); err != nil {
		sp.logger.Error("Failed to unmarshal charge event", zap.Error(err))
		return
	}

	chargeModel := new(stripe.Charge)
	if err := json.Unmarshal(eventModel.Data.Raw, &chargeModel); err != nil {
		sp.logger.Error("Failed to unmarshal charge event", zap.Error(err))
		return
	}

	partialCharge := &models.PartialCharge{
		ID: chargeModel.ID,
	}

	if chargeModel.Customer != nil {
		partialCharge.CustomerID = &chargeModel.Customer.ID
	}
	if chargeModel.PaymentIntent != nil {
		partialCharge.PaymentIntentID = &chargeModel.PaymentIntent.ID
	}
	if chargeModel.Amount > 0 {
		amount := float64(chargeModel.Amount) / 100
		partialCharge.Amount = &amount
	}
	if chargeModel.Currency != "" {
		partialCharge.Currency = &chargeModel.Currency
	}
	if chargeModel.Status != "" {
		partialCharge.Status = &chargeModel.Status
	}
	partialCharge.Paid = &chargeModel.Paid
	partialCharge.Refunded = &chargeModel.Refunded
	if chargeModel.FailureCode != "" {
		partialCharge.FailureCode = &chargeModel.FailureCode
	}
	if chargeModel.FailureMessage != "" {
		partialCharge.FailureMessage = &chargeModel.FailureMessage
	}
	if chargeModel.Created > 0 {
		createdAt := time.Unix(chargeModel.Created, 0)
		partialCharge.CreatedAt = &createdAt
	}

	if err := sp.charge.Upsert(ctx, partialCharge); err != nil {
		sp.logger.Error("Failed to upsert charge", zap.Error(err))
	}

	if err := sp.natsConn.Publish(NatsSubjectEventProcessed, msg.Data); err != nil {
		sp.logger.Error("Failed to publish event to NATS", zap.Error(err))
	}

	sp.logger.Info("Stripe charge event processed", zap.String("event_id", msg.Subject))
}

func (sp *StripePayment) handleRefundEvent(msg *nats.Msg) {

	sp.logger.Info("Stripe refund event received", zap.String("event_id", msg.Subject))

	ctx := context.Background()
	var eventModel stripe.Event
	if err := json.Unmarshal(msg.Data, &eventModel); err != nil {
		sp.logger.Error("Failed to unmarshal refund event", zap.Error(err))
		return
	}

	refundModel := new(stripe.Refund)
	if err := json.Unmarshal(eventModel.Data.Raw, &refundModel); err != nil {
		sp.logger.Error("Failed to unmarshal refund event", zap.Error(err))
		return
	}
	partialRefund := &models.PartialRefund{
		ID: refundModel.ID,
	}

	if refundModel.Charge != nil {
		partialRefund.ChargeID = &refundModel.Charge.ID
	}
	if refundModel.Amount > 0 {
		amount := float64(refundModel.Amount)
		partialRefund.Amount = &amount
	}
	if refundModel.Status != "" {
		partialRefund.Status = &refundModel.Status
	}
	if refundModel.Reason != "" {
		partialRefund.Reason = &refundModel.Reason
	}
	if refundModel.Created > 0 {
		createdAt := time.Unix(refundModel.Created, 0)
		partialRefund.CreatedAt = &createdAt
	}

	if err := sp.refund.Upsert(ctx, partialRefund); err != nil {
		sp.logger.Error(fmt.Sprintf("處理退款失敗: %s", err))
	}

	if err := sp.natsConn.Publish(NatsSubjectEventProcessed, msg.Data); err != nil {
		sp.logger.Error("Failed to publish event to NATS", zap.Error(err))
	}

	sp.logger.Info("Stripe refund event processed", zap.String("event_id", msg.Subject))
}

func (sp *StripePayment) handleDisputeEvent(msg *nats.Msg) {

	sp.logger.Info("Stripe dispute event received", zap.String("event_id", msg.Subject))

	ctx := context.Background()
	var eventModel stripe.Event
	if err := json.Unmarshal(msg.Data, &eventModel); err != nil {
		sp.logger.Error("Failed to unmarshal dispute event", zap.Error(err))
		return
	}

	dispute := new(stripe.Dispute)
	if err := json.Unmarshal(eventModel.Data.Raw, &dispute); err != nil {
		sp.logger.Error("Failed to unmarshal dispute event", zap.Error(err))
		return
	}
	partialDispute := &models.PartialDispute{
		ID: dispute.ID,
	}

	if dispute.Charge != nil {
		partialDispute.ChargeID = &dispute.Charge.ID
	}
	if dispute.Amount > 0 {
		partialDispute.Amount = &dispute.Amount
	}
	if dispute.Status != "" {
		partialDispute.Status = &dispute.Status
	}
	if dispute.Reason != "" {
		partialDispute.Reason = &dispute.Reason
	}
	if dispute.Created > 0 {
		createdAt := time.Unix(dispute.Created, 0)
		partialDispute.CreatedAt = &createdAt
	}

	var err error
	switch eventModel.Type {
	case "charge.dispute.created", "charge.dispute.updated":
		err = sp.dispute.Upsert(ctx, partialDispute)
	case "charge.dispute.closed":
		err = sp.dispute.Close(ctx, dispute.ID)
	default:
		sp.logger.Error(fmt.Sprintf("unexpected dispute event type: %s", eventModel.Type))
	}
	if err != nil {
		sp.logger.Error("Failed to upsert dispute", zap.Error(err))
	}

	if err = sp.natsConn.Publish(NatsSubjectEventProcessed, msg.Data); err != nil {
		sp.logger.Error("Failed to publish event to NATS", zap.Error(err))
	}

	sp.logger.Info("Stripe dispute event processed", zap.String("event_id", msg.Subject))
}

func (sp *StripePayment) handleProductEvent(msg *nats.Msg) {

	sp.logger.Info("Stripe product event received", zap.String("event_id", msg.Subject))

	ctx := context.Background()
	var eventModel stripe.Event
	if err := json.Unmarshal(msg.Data, &eventModel); err != nil {
		sp.logger.Error("Failed to unmarshal product event", zap.Error(err))
		return
	}

	productModel := new(stripe.Product)
	if err := json.Unmarshal(eventModel.Data.Raw, &productModel); err != nil {
		sp.logger.Error("Failed to unmarshal product event", zap.Error(err))
		return
	}

	partialProduct := &models.PartialProduct{
		ID: productModel.ID,
	}

	if productModel.Name != "" {
		partialProduct.Name = &productModel.Name
	}
	if productModel.Description != "" {
		partialProduct.Description = &productModel.Description
	}
	partialProduct.Active = &productModel.Active
	if productModel.Metadata != nil {
		partialProduct.Metadata = &productModel.Metadata
	}

	var err error
	switch eventModel.Type {
	case "product.created", "product.updated":
		err = sp.product.Upsert(ctx, partialProduct)
	case "product.deleted":
		err = sp.product.Delete(ctx, productModel.ID)
	default:
		sp.logger.Error(fmt.Sprintf("unexpected product event type: %s", eventModel.Type))
	}
	if err != nil {
		sp.logger.Error("Failed to upsert product", zap.Error(err))
	}

	if err = sp.natsConn.Publish(NatsSubjectEventProcessed, msg.Data); err != nil {
		sp.logger.Error("Failed to publish event to NATS", zap.Error(err))
	}

	sp.logger.Info("Stripe product event processed", zap.String("event_id", msg.Subject))
}

func (sp *StripePayment) handlePriceEvent(msg *nats.Msg) {

	sp.logger.Info("Stripe price event received", zap.String("event_id", msg.Subject))

	ctx := context.Background()
	var eventModel stripe.Event
	if err := json.Unmarshal(msg.Data, &eventModel); err != nil {
		sp.logger.Error("Failed to unmarshal price event", zap.Error(err))
		return
	}

	priceModel := new(stripe.Price)
	if err := json.Unmarshal(eventModel.Data.Raw, &priceModel); err != nil {
		sp.logger.Error("Failed to unmarshal price event", zap.Error(err))
		return
	}

	partialPrice := &models.PartialPrice{
		ID: priceModel.ID,
	}

	if priceModel.Product != nil {
		partialPrice.ProductID = &priceModel.Product.ID
	}
	partialPrice.Active = &priceModel.Active
	if priceModel.Currency != "" {
		partialPrice.Currency = &priceModel.Currency
	}
	if priceModel.UnitAmount > 0 {
		unitAmount := float64(priceModel.UnitAmount) / 100
		partialPrice.UnitAmount = &unitAmount
	}
	if priceModel.Type != "" {
		partialPrice.Type = &priceModel.Type
	}
	if priceModel.Recurring != nil {
		if priceModel.Recurring.Interval != "" {
			partialPrice.RecurringInterval = &priceModel.Recurring.Interval
		}
		if priceModel.Recurring.IntervalCount > 0 {
			intervalCount := int32(priceModel.Recurring.IntervalCount)
			partialPrice.RecurringIntervalCount = &intervalCount
		}
	}

	var err error
	switch eventModel.Type {
	case "price.created", "price.updated":
		err = sp.price.Upsert(ctx, partialPrice)
	case "price.deleted":
		err = sp.price.Delete(ctx, priceModel.ID)
	default:
		sp.logger.Error(fmt.Sprintf("unexpected price event type: %s", eventModel.Type))
	}

	if err != nil {
		sp.logger.Error("Failed to upsert price", zap.Error(err))
	}

	if err = sp.natsConn.Publish(NatsSubjectEventProcessed, msg.Data); err != nil {
		sp.logger.Error("Failed to publish event to NATS", zap.Error(err))
	}

	sp.logger.Info("Stripe price event processed", zap.String("event_id", msg.Subject))
}

func (sp *StripePayment) handlePaymentMethodEvent(msg *nats.Msg) {

	sp.logger.Info("Stripe payment method event received", zap.String("event_id", msg.Subject))

	ctx := context.Background()
	var eventModel stripe.Event
	if err := json.Unmarshal(msg.Data, &eventModel); err != nil {
		sp.logger.Error("Failed to unmarshal payment method event", zap.Error(err))
		return
	}

	paymentMethod := new(stripe.PaymentMethod)
	if err := json.Unmarshal(eventModel.Data.Raw, &paymentMethod); err != nil {
		sp.logger.Error("Failed to unmarshal payment method event", zap.Error(err))
		return
	}

	partialPaymentMethod := &models.PartialPaymentMethod{
		ID: paymentMethod.ID,
	}

	if paymentMethod.Customer != nil {
		partialPaymentMethod.CustomerID = &paymentMethod.Customer.ID
	}
	if paymentMethod.Type != "" {
		pmType := paymentMethod.Type
		partialPaymentMethod.Type = &pmType
	}
	if paymentMethod.Created > 0 {
		createdAt := time.Unix(paymentMethod.Created, 0)
		partialPaymentMethod.CreatedAt = &createdAt
	}

	switch paymentMethod.Type {
	case stripe.PaymentMethodTypeCard:
		if paymentMethod.Card != nil {
			if paymentMethod.Card.Last4 != "" {
				partialPaymentMethod.CardLast4 = &paymentMethod.Card.Last4
			}
			if paymentMethod.Card.Brand != "" {
				partialPaymentMethod.CardBrand = &paymentMethod.Card.Brand
			}
			if paymentMethod.Card.ExpMonth > 0 {
				expMonth := int32(paymentMethod.Card.ExpMonth)
				partialPaymentMethod.CardExpMonth = &expMonth
			}
			if paymentMethod.Card.ExpYear > 0 {
				expYear := int32(paymentMethod.Card.ExpYear)
				partialPaymentMethod.CardExpYear = &expYear
			}
		}
	case stripe.PaymentMethodTypeUSBankAccount:
		if paymentMethod.USBankAccount != nil {
			if paymentMethod.USBankAccount.Last4 != "" {
				partialPaymentMethod.BankAccountLast4 = &paymentMethod.USBankAccount.Last4
			}
			if paymentMethod.USBankAccount.BankName != "" {
				partialPaymentMethod.BankAccountBankName = &paymentMethod.USBankAccount.BankName
			}
		}
	}

	var err error
	switch eventModel.Type {
	case "payment_method.attached", "payment_method.updated":
		err = sp.paymentMethod.Upsert(ctx, partialPaymentMethod)
	case "payment_method.detached":
		err = sp.paymentMethod.Delete(ctx, paymentMethod.ID)
	default:
		sp.logger.Error(fmt.Sprintf("unexpected payment method event type: %s", eventModel.Type))
	}

	if err != nil {
		sp.logger.Error("Failed to upsert payment method", zap.Error(err))
	}

	if err = sp.natsConn.Publish(NatsSubjectEventProcessed, msg.Data); err != nil {
		sp.logger.Error("Failed to publish event to NATS", zap.Error(err))
	}

	sp.logger.Info("Stripe payment method event processed", zap.String("event_id", msg.Subject))
}

func (sp *StripePayment) handleCouponEvent(msg *nats.Msg) {

	sp.logger.Info("Stripe coupon event received", zap.String("event_id", msg.Subject))

	ctx := context.Background()
	var eventModel stripe.Event
	if err := json.Unmarshal(msg.Data, &eventModel); err != nil {
		sp.logger.Error("Failed to unmarshal coupon event", zap.Error(err))
		return
	}

	couponModel := new(stripe.Coupon)
	if err := json.Unmarshal(eventModel.Data.Raw, &couponModel); err != nil {
		sp.logger.Error("Failed to unmarshal coupon event", zap.Error(err))
		return
	}
	partialCoupon := &models.PartialCoupon{
		ID: couponModel.ID,
	}

	if couponModel.Name != "" {
		partialCoupon.Name = &couponModel.Name
	}
	if couponModel.Currency != "" {
		partialCoupon.Currency = &couponModel.Currency
	}
	if couponModel.Duration != "" {
		partialCoupon.Duration = &couponModel.Duration
	}
	timesRedeemed := int32(couponModel.TimesRedeemed)
	partialCoupon.TimesRedeemed = &timesRedeemed
	partialCoupon.Valid = &couponModel.Valid
	if couponModel.Created > 0 {
		createdAt := time.Unix(couponModel.Created, 0)
		partialCoupon.CreatedAt = &createdAt
	}

	if couponModel.AmountOff > 0 {
		partialCoupon.AmountOff = &couponModel.AmountOff
	}
	if couponModel.PercentOff > 0 {
		partialCoupon.PercentOff = &couponModel.PercentOff
	}
	if couponModel.DurationInMonths > 0 {
		durationInMonths := int(couponModel.DurationInMonths)
		partialCoupon.DurationInMonths = &durationInMonths
	}
	if couponModel.MaxRedemptions > 0 {
		maxRedemptions := int(couponModel.MaxRedemptions)
		partialCoupon.MaxRedemptions = &maxRedemptions
	}
	if couponModel.RedeemBy > 0 {
		redeemBy := time.Unix(couponModel.RedeemBy, 0)
		partialCoupon.RedeemBy = &redeemBy
	}

	var err error
	switch eventModel.Type {
	case "coupon.created", "coupon.updated":
		err = sp.coupon.Upsert(ctx, partialCoupon)
	case "coupon.deleted":
		err = sp.coupon.Delete(ctx, couponModel.ID)
	default:
		sp.logger.Error(fmt.Sprintf("unexpected coupon event type: %s", eventModel.Type))
	}

	if err != nil {
		sp.logger.Error("Failed to upsert coupon", zap.Error(err))
	}

	if err = sp.natsConn.Publish(NatsSubjectEventProcessed, msg.Data); err != nil {
		sp.logger.Error("Failed to publish event to NATS", zap.Error(err))
	}

	sp.logger.Info("Stripe coupon event processed", zap.String("event_id", msg.Subject))
}

func (sp *StripePayment) handleDiscountEvent(msg *nats.Msg) {

	sp.logger.Info("Stripe discount event received", zap.String("event_id", msg.Subject))

	ctx := context.Background()
	var eventModel stripe.Event
	if err := json.Unmarshal(msg.Data, &eventModel); err != nil {
		sp.logger.Error("Failed to unmarshal discount event", zap.Error(err))
		return
	}

	discountModel := new(stripe.Discount)
	if err := json.Unmarshal(eventModel.Data.Raw, &discountModel); err != nil {
		sp.logger.Error("Failed to unmarshal discount event", zap.Error(err))
		return
	}

	partialDiscount := &models.PartialDiscount{
		ID: discountModel.ID,
	}

	if discountModel.Customer != nil {
		partialDiscount.CustomerID = &discountModel.Customer.ID
	}
	if discountModel.Coupon != nil {
		partialDiscount.CouponID = &discountModel.Coupon.ID
	}
	if discountModel.Start > 0 {
		start := time.Unix(discountModel.Start, 0)
		partialDiscount.Start = &start
		partialDiscount.CreatedAt = &start
	}
	if discountModel.End > 0 {
		end := time.Unix(discountModel.End, 0)
		partialDiscount.End = &end
	}

	var err error
	switch eventModel.Type {
	case "customer.discount.created", "customer.discount.updated":
		err = sp.discount.Upsert(ctx, partialDiscount)
	case "customer.discount.deleted":
		err = sp.discount.Delete(ctx, discountModel.ID)
	default:
		sp.logger.Error(fmt.Sprintf("unexpected discount event type: %s", eventModel.Type))
	}

	if err != nil {
		sp.logger.Error("Failed to upsert discount object", zap.Error(err))
	}

	if err = sp.natsConn.Publish(NatsSubjectEventProcessed, msg.Data); err != nil {
		sp.logger.Error("Failed to publish event to NATS", zap.Error(err))
	}

	sp.logger.Info("Stripe discount event processed", zap.String("event_id", msg.Subject))
}

func (sp *StripePayment) handlePromotionCodeEvent(msg *nats.Msg) {

	sp.logger.Info("Stripe promotion code event received", zap.String("event_id", msg.Subject))

	ctx := context.Background()
	var eventModel stripe.Event
	if err := json.Unmarshal(msg.Data, &eventModel); err != nil {
		sp.logger.Error("Failed to unmarshal promotion code event", zap.Error(err))
		return
	}

	promotionCode := new(stripe.PromotionCode)
	if err := json.Unmarshal(eventModel.Data.Raw, &promotionCode); err != nil {
		sp.logger.Error("Failed to unmarshal promotion code event", zap.Error(err))
		return
	}
	partialPromotionCode := &models.PartialPromotionCode{
		ID: promotionCode.ID,
	}

	if promotionCode.Code != "" {
		partialPromotionCode.Code = &promotionCode.Code
	}
	if promotionCode.Coupon != nil {
		partialPromotionCode.CouponID = &promotionCode.Coupon.ID
	}
	if promotionCode.Customer != nil {
		partialPromotionCode.CustomerID = &promotionCode.Customer.ID
	}
	partialPromotionCode.Active = &promotionCode.Active
	if promotionCode.MaxRedemptions > 0 {
		maxRedemptions := int(promotionCode.MaxRedemptions)
		partialPromotionCode.MaxRedemptions = &maxRedemptions
	}
	timesRedeemed := int(promotionCode.TimesRedeemed)
	partialPromotionCode.TimesRedeemed = &timesRedeemed
	if promotionCode.ExpiresAt > 0 {
		expiresAt := time.Unix(promotionCode.ExpiresAt, 0)
		partialPromotionCode.ExpiresAt = &expiresAt
	}
	if promotionCode.Created > 0 {
		createdAt := time.Unix(promotionCode.Created, 0)
		partialPromotionCode.CreatedAt = &createdAt
	}

	var err error
	switch eventModel.Type {
	case "promotion_code.created", "promotion_code.updated":
		err = sp.promotionCode.Upsert(ctx, partialPromotionCode)
	default:
		sp.logger.Error(fmt.Sprintf("unexpected promotionCode event type: %s", eventModel.Type))
	}
	if err != nil {
		sp.logger.Error("Failed to upsert promotion code object", zap.Error(err))
	}

	if err = sp.natsConn.Publish(NatsSubjectEventProcessed, msg.Data); err != nil {
		sp.logger.Error("Failed to publish event to NATS", zap.Error(err))
	}

	sp.logger.Info("Stripe promotion code event processed", zap.String("event_id", msg.Subject))
}

func (sp *StripePayment) handleCheckoutSessionEvent(msg *nats.Msg) {

	sp.logger.Info("Stripe checkout session event received", zap.String("event_id", msg.Subject))

	ctx := context.Background()
	var eventModel stripe.Event
	if err := json.Unmarshal(msg.Data, &eventModel); err != nil {
		sp.logger.Error("Failed to unmarshal checkout session event", zap.Error(err))
		return
	}

	session := new(stripe.CheckoutSession)
	if err := json.Unmarshal(eventModel.Data.Raw, &session); err != nil {
		sp.logger.Error("Failed to unmarshal checkout session event", zap.Error(err))
		return
	}
	partialSession := &models.PartialCheckoutSession{
		ID: session.ID,
	}

	if session.Customer != nil {
		partialSession.CustomerID = &session.Customer.ID
	}
	if session.PaymentIntent != nil {
		partialSession.PaymentIntentID = &session.PaymentIntent.ID
	}
	partialSession.Status = &session.Status
	partialSession.Mode = &session.Mode
	partialSession.SuccessURL = &session.SuccessURL
	partialSession.CancelURL = &session.CancelURL
	amountTotal := session.AmountTotal
	partialSession.AmountTotal = &amountTotal
	partialSession.Currency = &session.Currency
	if session.Created > 0 {
		createdAt := time.Unix(session.Created, 0)
		partialSession.CreatedAt = &createdAt
	}

	var err error
	switch eventModel.Type {
	case "checkout.session.completed", "checkout.session.async_payment_succeeded",
		"checkout.session.async_payment_failed", "checkout.session.expired":
		err = sp.checkoutSession.Upsert(ctx, partialSession)
	default:
		sp.logger.Error(fmt.Sprintf("unexpected checkoutSession event type: %s", eventModel.Type))
	}

	if err != nil {
		sp.logger.Error("Failed to upsert checkout session object", zap.Error(err))
	}

	if err = sp.natsConn.Publish(NatsSubjectEventProcessed, msg.Data); err != nil {
		sp.logger.Error("Failed to publish event to NATS", zap.Error(err))
	}

	sp.logger.Info("Stripe checkout session event processed", zap.String("event_id", msg.Subject))
}

func (sp *StripePayment) handleQuoteEvent(msg *nats.Msg) {

	sp.logger.Info("Stripe quote event received", zap.String("event_id", msg.Subject))

	ctx := context.Background()
	var eventModel stripe.Event
	if err := json.Unmarshal(msg.Data, &eventModel); err != nil {
		sp.logger.Error("Failed to unmarshal quote event", zap.Error(err))
		return
	}
	quoteModel := new(stripe.Quote)
	if err := json.Unmarshal(eventModel.Data.Raw, &quoteModel); err != nil {
		sp.logger.Error("Failed to unmarshal quote event", zap.Error(err))
		return
	}

	partialQuote := &models.PartialQuote{
		ID: quoteModel.ID,
	}

	if quoteModel.Customer != nil {
		partialQuote.CustomerID = &quoteModel.Customer.ID
	}
	partialQuote.Status = &quoteModel.Status

	amountTotal := quoteModel.AmountTotal
	partialQuote.AmountTotal = &amountTotal

	partialQuote.Currency = &quoteModel.Currency

	if quoteModel.ExpiresAt > 0 {
		validUntil := time.Unix(quoteModel.ExpiresAt, 0)
		partialQuote.ValidUntil = &validUntil
	}

	// Stripe的Quote模型中沒有直接的AcceptedAt字段，
	// 從StatusTransitions中獲取，如果存在的話
	if quoteModel.StatusTransitions != nil && quoteModel.StatusTransitions.AcceptedAt > 0 {
		acceptedAt := time.Unix(quoteModel.StatusTransitions.AcceptedAt, 0)
		partialQuote.AcceptedAt = &acceptedAt
	}

	// CanceledAt也可以從StatusTransitions中獲取
	if quoteModel.StatusTransitions != nil && quoteModel.StatusTransitions.CanceledAt > 0 {
		canceledAt := time.Unix(quoteModel.StatusTransitions.CanceledAt, 0)
		partialQuote.CanceledAt = &canceledAt
	}

	if quoteModel.Created > 0 {
		createdAt := time.Unix(quoteModel.Created, 0)
		partialQuote.CreatedAt = &createdAt
	}

	var err error
	switch eventModel.Type {
	case "quote.created", "quote.finalized", "quote.accepted", "quote.canceled":
		err = sp.quote.Upsert(ctx, partialQuote)
	default:
		sp.logger.Error(fmt.Sprintf("unexpected quote event type: %s", eventModel.Type))
	}

	if err != nil {
		sp.logger.Error("Failed to upsert quote object", zap.Error(err))
	}

	if err = sp.natsConn.Publish(NatsSubjectEventProcessed, msg.Data); err != nil {
		sp.logger.Error("Failed to publish event to NATS", zap.Error(err))
	}

	sp.logger.Info("Stripe quote event processed", zap.String("event_id", msg.Subject))
}

func (sp *StripePayment) handlePaymentLinkEvent(msg *nats.Msg) {

	sp.logger.Info("Stripe payment link event received", zap.String("event_id", msg.Subject))

	ctx := context.Background()
	var eventModel stripe.Event
	if err := json.Unmarshal(msg.Data, &eventModel); err != nil {
		sp.logger.Error("Failed to unmarshal payment link event", zap.Error(err))
		return
	}
	paymentLink := new(stripe.PaymentLink)
	if err := json.Unmarshal(eventModel.Data.Raw, &paymentLink); err != nil {
		sp.logger.Error("Failed to unmarshal payment link event", zap.Error(err))
		return
	}
	partialPaymentLink := &models.PartialPaymentLink{
		ID: paymentLink.ID,
	}

	partialPaymentLink.Active = &paymentLink.Active
	partialPaymentLink.URL = &paymentLink.URL

	if paymentLink.LineItems != nil && len(paymentLink.LineItems.Data) > 0 {
		totalAmount := int64(0)
		for _, item := range paymentLink.LineItems.Data {
			totalAmount += item.AmountSubtotal
		}
		partialPaymentLink.Amount = &totalAmount
	}

	partialPaymentLink.Currency = &paymentLink.Currency

	var err error
	switch eventModel.Type {
	case "payment_link.created", "payment_link.updated":
		err = sp.paymentLink.Upsert(ctx, partialPaymentLink)
	default:
		sp.logger.Error(fmt.Sprintf("unexpected payment link event type: %s", eventModel.Type))
	}

	if err != nil {
		sp.logger.Error("Failed to upsert payment link object", zap.Error(err))
	}

	if err = sp.natsConn.Publish(NatsSubjectEventProcessed, msg.Data); err != nil {
		sp.logger.Error("Failed to publish event to NATS", zap.Error(err))
	}

	sp.logger.Info("Stripe payment link event processed", zap.String("event_id", msg.Subject))
}

func (sp *StripePayment) handleTaxRateEvent(msg *nats.Msg) {

	sp.logger.Info("Stripe tax rate event received", zap.String("event_id", msg.Subject))

	ctx := context.Background()
	var eventModel stripe.Event
	if err := json.Unmarshal(msg.Data, &eventModel); err != nil {
		sp.logger.Error("Failed to unmarshal tax rate event", zap.Error(err))
		return
	}
	taxRate := new(stripe.TaxRate)
	if err := json.Unmarshal(eventModel.Data.Raw, &taxRate); err != nil {
		sp.logger.Error("Failed to unmarshal tax rate link event", zap.Error(err))
		return
	}
	partialTaxRate := &models.PartialTaxRate{
		ID: taxRate.ID,
	}

	partialTaxRate.DisplayName = &taxRate.DisplayName
	if taxRate.Description != "" {
		partialTaxRate.Description = &taxRate.Description
	}
	if taxRate.Jurisdiction != "" {
		partialTaxRate.Jurisdiction = &taxRate.Jurisdiction
	}
	percentage := taxRate.Percentage
	partialTaxRate.Percentage = &percentage
	partialTaxRate.Inclusive = &taxRate.Inclusive
	partialTaxRate.Active = &taxRate.Active
	if taxRate.Created > 0 {
		createdAt := time.Unix(taxRate.Created, 0)
		partialTaxRate.CreatedAt = &createdAt
	}

	var err error
	switch eventModel.Type {
	case "tax_rate.created", "tax_rate.updated":
		err = sp.taxRate.Upsert(ctx, partialTaxRate)
	default:
		sp.logger.Error(fmt.Sprintf("unexpected tax rate event type: %s", eventModel.Type))
	}

	if err != nil {
		sp.logger.Error("Failed to upsert tax rate object", zap.Error(err))
	}

	if err = sp.natsConn.Publish(NatsSubjectEventProcessed, msg.Data); err != nil {
		sp.logger.Error("Failed to publish event to NATS", zap.Error(err))
	}

	sp.logger.Info("Stripe tax rate event processed", zap.String("event_id", msg.Subject))

}

func (sp *StripePayment) handleReviewEvent(msg *nats.Msg) {

	sp.logger.Info("Stripe review event received", zap.String("event_id", msg.Subject))

	ctx := context.Background()
	var eventModel stripe.Event
	if err := json.Unmarshal(msg.Data, &eventModel); err != nil {
		sp.logger.Error("Failed to unmarshal review event", zap.Error(err))
		return
	}
	reviewModel := new(stripe.Review)
	if err := json.Unmarshal(eventModel.Data.Raw, &reviewModel); err != nil {
		sp.logger.Error("Failed to unmarshal review event", zap.Error(err))
		return
	}
	partialReview := &models.PartialReview{
		ID: reviewModel.ID,
	}

	if reviewModel.PaymentIntent != nil {
		partialReview.PaymentIntentID = &reviewModel.PaymentIntent.ID
	}
	partialReview.Reason = &reviewModel.Reason

	// 根據 Open 字段設置狀態
	var status string
	if reviewModel.Open {
		status = "open"
	} else {
		status = "closed"
	}
	partialReview.Status = &status

	if reviewModel.Created > 0 {
		createdAt := time.Unix(reviewModel.Created, 0)
		partialReview.CreatedAt = &createdAt
		// 使用 Created 時間作為 OpenedAt
		partialReview.OpenedAt = &createdAt
	}

	// 如果 Review 已關閉，設置 ClosedAt
	if !reviewModel.Open {
		closedAt := time.Now()
		partialReview.ClosedAt = &closedAt
	}

	// ClosedReason
	if reviewModel.ClosedReason != "" {
		partialReview.ClosedReason = &reviewModel.ClosedReason
	}

	var err error
	switch eventModel.Type {
	case "review.opened", "review.closed":
		err = sp.review.Upsert(ctx, partialReview)
	default:
		sp.logger.Error(fmt.Sprintf("unexpected review event type: %s", eventModel.Type))
	}

	if err != nil {
		sp.logger.Error("Failed to upsert review object", zap.Error(err))
	}

	if err = sp.natsConn.Publish(NatsSubjectEventProcessed, msg.Data); err != nil {
		sp.logger.Error("Failed to publish event to NATS", zap.Error(err))
	}

	sp.logger.Info("Stripe review event processed", zap.String("event_id", msg.Subject))

}

func (sp *StripePayment) Close() {
	sp.logger.Info("Initiating graceful shutdown of workers and dispatcher")
	for _, sub := range sp.subscriptions {
		if sub != nil {
			if err := sub.Unsubscribe(); err != nil {
				sp.logger.Error("Failed to unsubscribe", zap.Error(err))
			}
		}
	}
	sp.natsConn.Close()
	sp.logger.Info("StripePayment successfully shutdown")
}
