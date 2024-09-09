package payment

import (
	"context"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"goflare.io/payment/charge"
	"goflare.io/payment/coupon"
	"goflare.io/payment/discount"
	"goflare.io/payment/disputes"
	"goflare.io/payment/event"
	"golang.org/x/sync/errgroup"
	"strconv"
	"sync"
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
	charge               charge.Service
	coupon               coupon.Service
	customerService      customer.Service
	discount             discount.Service
	dispute              disputes.Service
	event                event.Service
	productService       product.Service
	priceService         price.Service
	subscriptionService  subscription.Service
	invoiceService       invoice.Service
	paymentMethodService payment_method.Service
	paymentIntentService payment_intent.Service
	refundService        refund.Service

	eventChan   chan *stripe.Event
	workerCount int
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	logger      *zap.Logger
}

func NewStripePayment(config *config.Config,
	cs customer.Service,
	charge charge.Service,
	coupon coupon.Service,
	discount discount.Service,
	dispute disputes.Service,
	event event.Service,
	ps product.Service,
	prs price.Service,
	ss subscription.Service,
	is invoice.Service,
	pms payment_method.Service,
	pis payment_intent.Service,
	rs refund.Service,
	logger *zap.Logger) Payment {
	ctx, cancel := context.WithCancel(context.Background())
	sp := &StripePayment{
		client:               client.New(config.Stripe.SecretKey, nil),
		charge:               charge,
		coupon:               coupon,
		customerService:      cs,
		discount:             discount,
		dispute:              dispute,
		event:                event,
		productService:       ps,
		priceService:         prs,
		subscriptionService:  ss,
		invoiceService:       is,
		paymentMethodService: pms,
		paymentIntentService: pis,
		refundService:        rs,

		eventChan:   make(chan *stripe.Event, 100), // 緩衝區大小可以根據需求調整
		workerCount: 10,                            // 可以根據需求調整
		ctx:         ctx,
		cancel:      cancel,
		logger:      logger,
	}

	sp.startWorkers()

	return sp
}

func (sp *StripePayment) startWorkers() {
	sp.wg.Add(sp.workerCount)
	for i := 0; i < sp.workerCount; i++ {
		go sp.eventWorker()
	}
}

func (sp *StripePayment) eventWorker() {
	defer sp.wg.Done()
	for {
		select {
		case e, ok := <-sp.eventChan:
			if !ok {
				return
			}
			if err := sp.processEvent(sp.ctx, e); err != nil {
				sp.logger.Error("Error processing event",
					zap.Error(err),
					zap.String("event_type", string(e.Type)),
					zap.String("event_id", e.ID))
			}
		case <-sp.ctx.Done():
			return
		}
	}
}

func (sp *StripePayment) processEvent(ctx context.Context, event *stripe.Event) error {
	// 使用 errgroup 來管理可能的並發操作
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var handleErr error
		start := time.Now()

		switch event.Type {
		case "customer.created", "customer.updated", "customer.deleted":
			var customerModel stripe.Customer
			if err := json.Unmarshal(event.Data.Raw, &customerModel); err != nil {
				return fmt.Errorf("failed to unmarshal customer data: %w", err)
			}
			handleErr = sp.handleCustomerEvent(ctx, &customerModel, event.Type)

		case "customer.subscription.created", "customer.subscription.updated", "customer.subscription.deleted":
			var subscriptionModel stripe.Subscription
			if err := json.Unmarshal(event.Data.Raw, &subscriptionModel); err != nil {
				return fmt.Errorf("failed to unmarshal subscription data: %w", err)
			}
			handleErr = sp.handleSubscriptionEvent(ctx, &subscriptionModel, event.Type)

		case "invoice.created", "invoice.updated", "invoice.paid", "invoice.payment_failed":
			var invoiceModel stripe.Invoice
			if err := json.Unmarshal(event.Data.Raw, &invoiceModel); err != nil {
				return fmt.Errorf("failed to unmarshal invoice data: %w", err)
			}
			handleErr = sp.handleInvoiceEvent(ctx, &invoiceModel, event.Type)

		case "payment_intent.created", "payment_intent.succeeded", "payment_intent.payment_failed", "payment_intent.canceled":
			var paymentIntentModel stripe.PaymentIntent
			if err := json.Unmarshal(event.Data.Raw, &paymentIntentModel); err != nil {
				return fmt.Errorf("failed to unmarshal payment intent data: %w", err)
			}
			handleErr = sp.handlePaymentIntentEvent(ctx, &paymentIntentModel)

		case "charge.succeeded", "charge.failed", "charge.refunded":
			var chargeModel stripe.Charge
			if err := json.Unmarshal(event.Data.Raw, &chargeModel); err != nil {
				return fmt.Errorf("failed to unmarshal charge data: %w", err)
			}
			handleErr = sp.handleChargeEvent(ctx, &chargeModel)

		case "charge.dispute.created", "charge.dispute.updated", "charge.dispute.closed":
			var disputeModel stripe.Dispute
			if err := json.Unmarshal(event.Data.Raw, &disputeModel); err != nil {
				return fmt.Errorf("failed to unmarshal dispute data: %w", err)
			}
			handleErr = sp.handleDisputeEvent(ctx, &disputeModel, event.Type)

		case "product.created", "product.updated", "product.deleted":
			var productModel stripe.Product
			if err := json.Unmarshal(event.Data.Raw, &productModel); err != nil {
				return fmt.Errorf("failed to unmarshal product data: %w", err)
			}
			handleErr = sp.handleProductEvent(ctx, &productModel, event.Type)

		case "price.created", "price.updated", "price.deleted":
			var priceModel stripe.Price
			if err := json.Unmarshal(event.Data.Raw, &priceModel); err != nil {
				return fmt.Errorf("failed to unmarshal price data: %w", err)
			}
			handleErr = sp.handlePriceEvent(ctx, &priceModel, event.Type)

		case "payment_method.attached", "payment_method.updated", "payment_method.detached":
			var paymentMethodModel stripe.PaymentMethod
			if err := json.Unmarshal(event.Data.Raw, &paymentMethodModel); err != nil {
				return fmt.Errorf("failed to unmarshal payment method data: %w", err)
			}
			handleErr = sp.handlePaymentMethodEvent(ctx, &paymentMethodModel, event.Type)

		case "coupon.created", "coupon.updated", "coupon.deleted":
			var couponModel stripe.Coupon
			if err := json.Unmarshal(event.Data.Raw, &couponModel); err != nil {
				return fmt.Errorf("failed to unmarshal coupon data: %w", err)
			}
			handleErr = sp.handleCouponEvent(ctx, &couponModel, event.Type)

		case "customer.discount.created", "customer.discount.updated", "customer.discount.deleted":
			var discountModel stripe.Discount
			if err := json.Unmarshal(event.Data.Raw, &discountModel); err != nil {
				return fmt.Errorf("failed to unmarshal discount data: %w", err)
			}
			handleErr = sp.handleDiscountEvent(ctx, &discountModel, event.Type)

		default:
			sp.logger.Info("Unhandled event type",
				zap.String("type", string(event.Type)),
				zap.String("event_id", event.ID))
			return nil
		}

		if handleErr != nil {
			return fmt.Errorf("error handling event %s: %w", event.Type, handleErr)
		}

		duration := time.Since(start)
		sp.logger.Info("Event processed successfully",
			zap.String("event_type", string(event.Type)),
			zap.String("event_id", event.ID),
			zap.Duration("duration", duration))

		return nil
	})

	// 等待所有操作完成
	if err := g.Wait(); err != nil {
		return err
	}

	// 標記事件為已處理
	if err := sp.event.MarkEventAsProcessed(ctx, event.ID); err != nil {
		return fmt.Errorf("failed to mark event as processed: %w", err)
	}

	return nil
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
		ID:     stripeCustomer.ID,
		UserID: userID,
		Name:   name,
		Email:  email,
	}
	if err = sp.customerService.Create(ctx, customerModel); err != nil {
		return fmt.Errorf("failed to create local customer record: %w", err)
	}

	return nil
}

// GetCustomer retrieves a customer from the local database
func (sp *StripePayment) GetCustomer(ctx context.Context, customerID string) (*models.Customer, error) {
	return sp.customerService.GetByID(ctx, customerID)
}

// UpdateCustomerBalance updates a customer in Stripe and in the local database
func (sp *StripePayment) UpdateCustomerBalance(ctx context.Context, updateCustomer *models.Customer) error {

	params := &stripe.CustomerParams{
		Balance: &updateCustomer.Balance,
	}

	if _, err := sp.client.Customers.Update(updateCustomer.ID, params); err != nil {
		return fmt.Errorf("failed to update Stripe customer: %w", err)
	}

	if err := sp.customerService.UpdateBalance(ctx, updateCustomer.ID, uint64(updateCustomer.Balance)); err != nil {
		return fmt.Errorf("failed to update local customer record: %w", err)
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
func (sp *StripePayment) GetProductWithAllPrices(ctx context.Context, productID string) (*models.Product, error) {
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
	return sp.productService.List(ctx, 1000, 0) // Assuming a large limit, you might want to implement pagination
}

// CreatePrice creates a new price in Stripe and in the local database
func (sp *StripePayment) CreatePrice(price models.Price) error {

	params := &stripe.PriceParams{
		Product:    stripe.String(price.ProductID),
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
	return sp.subscriptionService.GetByID(ctx, subscriptionID)
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
	return sp.subscriptionService.List(ctx, customerID, 1000, 0)
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
	return sp.invoiceService.GetByID(ctx, invoiceID)
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
	return sp.invoiceService.List(ctx, customerID, 1000, 0)
	// Assuming a large limit, you might want to implement pagination
}

// GetPaymentMethod retrieves a payment method from the local database
func (sp *StripePayment) GetPaymentMethod(ctx context.Context, paymentMethodID string) (*models.PaymentMethod, error) {
	return sp.paymentMethodService.GetByID(ctx, paymentMethodID)
}

// DeletePaymentMethod deletes a payment method from Stripe and from the local database
func (sp *StripePayment) DeletePaymentMethod(ctx context.Context, paymentMethodID string) error {

	if _, err := sp.client.PaymentMethods.Detach(paymentMethodID, nil); err != nil {
		return fmt.Errorf("failed to detach Stripe payment method: %w", err)
	}

	if err := sp.paymentMethodService.Delete(ctx, paymentMethodID); err != nil {
		return fmt.Errorf("failed to delete local payment method record: %w", err)
	}

	return nil
}

// ListPaymentMethods lists all payment methods for a customer from the local database
func (sp *StripePayment) ListPaymentMethods(ctx context.Context, customerID string) ([]*models.PaymentMethod, error) {
	return sp.paymentMethodService.List(ctx, customerID, 1000, 0)
	// Assuming a large limit, you might want to implement pagination
}

// CreatePaymentIntent creates a new payment intent in Stripe and in the local database
func (sp *StripePayment) CreatePaymentIntent(customerID, paymentMethodID string, amount uint64, currency enum.Currency) error {

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
	return sp.paymentIntentService.GetByID(ctx, paymentIntentID)
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
	return sp.paymentIntentService.List(ctx, limit, offset)
}

func (sp *StripePayment) ListPaymentIntentByCustomerID(ctx context.Context, customerID string, limit, offset uint64) ([]*models.PaymentIntent, error) {
	return sp.paymentIntentService.ListByCustomer(ctx, customerID, limit, offset)
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
	return sp.refundService.GetByID(ctx, refundID)
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
	return sp.refundService.ListByChargeID(ctx, chargeID)
}

// HandleStripeWebhook handles Stripe webhook events
func (sp *StripePayment) HandleStripeWebhook(ctx context.Context, payload []byte, signature string) error {
	stripeEvent, err := webhook.ConstructEvent(payload, signature, "")
	if err != nil {
		return fmt.Errorf("failed to verify webhook signature: %w", err)
	}

	// 檢查事件是否已經處理過
	processed, err := sp.event.IsEventProcessed(ctx, stripeEvent.ID)
	if err != nil {
		return fmt.Errorf("failed to check event status: %w", err)
	}
	if processed {
		// 事件已處理，直接返回
		return nil
	}

	// 使用 select 語句來處理通道可能已滿的情況
	select {
	case sp.eventChan <- &stripeEvent:
		// 事件成功加入處理隊列
	case <-time.After(5 * time.Second):
		// 如果 5 秒內無法將事件加入隊列，記錄錯誤並返回
		sp.logger.Error("Failed to queue event for processing",
			zap.String("event_id", stripeEvent.ID),
			zap.String("event_type", string(stripeEvent.Type)))
		return fmt.Errorf("event queue is full, unable to process event %s", stripeEvent.ID)
	}

	return nil
}

func (sp *StripePayment) handleCustomerEvent(ctx context.Context, customer *stripe.Customer, eventType stripe.EventType) error {
	partialCustomer := &models.PartialCustomer{
		ID: customer.ID,
	}

	if customer.Email != "" {
		partialCustomer.Email = &customer.Email
	}
	if customer.Name != "" {
		partialCustomer.Name = &customer.Name
	}
	if customer.Phone != "" {
		partialCustomer.Phone = &customer.Phone
	}
	if customer.Balance != 0 {
		balance := customer.Balance
		partialCustomer.Balance = &balance
	}
	if customer.Created > 0 {
		createdAt := time.Unix(customer.Created, 0)
		partialCustomer.CreatedAt = &createdAt
	}

	switch eventType {
	case "customer.created", "customer.updated":
		return sp.customerService.Upsert(ctx, partialCustomer)
	case "customer.deleted":
		return sp.customerService.Delete(ctx, customer.ID)
	default:
		return fmt.Errorf("unexpected customer event type: %s", eventType)
	}
}

func (sp *StripePayment) handleSubscriptionEvent(ctx context.Context, subscription *stripe.Subscription, eventType stripe.EventType) error {
	partialSubscription := &models.PartialSubscription{
		ID: subscription.ID,
	}

	if subscription.Customer != nil {
		partialSubscription.CustomerID = &subscription.Customer.ID
	}
	if subscription.Status != "" {
		status := enum.SubscriptionStatus(subscription.Status)
		partialSubscription.Status = &status
	}
	if subscription.CurrentPeriodStart > 0 {
		start := time.Unix(subscription.CurrentPeriodStart, 0)
		partialSubscription.CurrentPeriodStart = &start
	}
	if subscription.CurrentPeriodEnd > 0 {
		end := time.Unix(subscription.CurrentPeriodEnd, 0)
		partialSubscription.CurrentPeriodEnd = &end
	}
	partialSubscription.CancelAtPeriodEnd = &subscription.CancelAtPeriodEnd
	if subscription.CanceledAt > 0 {
		canceledAt := time.Unix(subscription.CanceledAt, 0)
		partialSubscription.CanceledAt = &canceledAt
	}

	switch eventType {
	case "customer.subscription.created", "customer.subscription.updated",
		"customer.subscription.trial_will_end", "customer.subscription.pending_update_applied",
		"customer.subscription.pending_update_expired", "customer.subscription.paused",
		"customer.subscription.resumed":
		return sp.subscriptionService.Upsert(ctx, partialSubscription)
	case "customer.subscription.deleted":
		return sp.subscriptionService.Delete(ctx, subscription.ID)
	default:
		return fmt.Errorf("unexpected subscription event type: %s", eventType)
	}
}

func (sp *StripePayment) handleInvoiceEvent(ctx context.Context, invoice *stripe.Invoice, eventType stripe.EventType) error {
	partialInvoice := &models.PartialInvoice{
		ID: invoice.ID,
	}

	if invoice.Customer != nil {
		partialInvoice.CustomerID = &invoice.Customer.ID
	}
	if invoice.Subscription != nil {
		partialInvoice.SubscriptionID = &invoice.Subscription.ID
	}
	if invoice.Status != "" {
		status := enum.InvoiceStatus(invoice.Status)
		partialInvoice.Status = &status
	}
	if invoice.Currency != "" {
		currency := enum.Currency(invoice.Currency)
		partialInvoice.Currency = &currency
	}
	if invoice.AmountDue > 0 {
		amountDue := float64(invoice.AmountDue) / 100
		partialInvoice.AmountDue = &amountDue
	}
	if invoice.AmountPaid > 0 {
		amountPaid := float64(invoice.AmountPaid) / 100
		partialInvoice.AmountPaid = &amountPaid
	}
	if invoice.AmountRemaining > 0 {
		amountRemaining := float64(invoice.AmountRemaining) / 100
		partialInvoice.AmountRemaining = &amountRemaining
	}
	if invoice.DueDate > 0 {
		dueDate := time.Unix(invoice.DueDate, 0)
		partialInvoice.DueDate = &dueDate
	}
	if invoice.Created > 0 {
		createdAt := time.Unix(invoice.Created, 0)
		partialInvoice.CreatedAt = &createdAt
	}

	if invoice.Status == stripe.InvoiceStatusPaid {
		now := time.Now()
		partialInvoice.PaidAt = &now
	}

	switch eventType {
	case "invoice.created", "invoice.updated", "invoice.finalized",
		"invoice.payment_succeeded", "invoice.payment_failed", "invoice.sent":
		return sp.invoiceService.Upsert(ctx, partialInvoice)
	case "invoice.deleted":
		return sp.invoiceService.Delete(ctx, invoice.ID)
	default:
		return fmt.Errorf("unexpected invoice event type: %s", eventType)
	}
}

func (sp *StripePayment) handlePaymentIntentEvent(ctx context.Context, paymentIntent *stripe.PaymentIntent) error {
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
		currency := enum.Currency(paymentIntent.Currency)
		partialPaymentIntent.Currency = &currency
	}
	if paymentIntent.Status != "" {
		status := enum.PaymentIntentStatus(paymentIntent.Status)
		partialPaymentIntent.Status = &status
	}
	if paymentIntent.PaymentMethod != nil {
		partialPaymentIntent.PaymentMethodID = &paymentIntent.PaymentMethod.ID
	}
	if paymentIntent.SetupFutureUsage != "" {
		setupFutureUsage := string(paymentIntent.SetupFutureUsage)
		partialPaymentIntent.SetupFutureUsage = &setupFutureUsage
	}
	if paymentIntent.ClientSecret != "" {
		partialPaymentIntent.ClientSecret = &paymentIntent.ClientSecret
	}
	if paymentIntent.CaptureMethod != "" {
		captureMethod := string(paymentIntent.CaptureMethod)
		partialPaymentIntent.CaptureMethod = &captureMethod
	}
	if paymentIntent.Created > 0 {
		createdAt := time.Unix(paymentIntent.Created, 0)
		partialPaymentIntent.CreatedAt = &createdAt
	}

	return sp.paymentIntentService.Upsert(ctx, partialPaymentIntent)
}

func (sp *StripePayment) handleChargeEvent(ctx context.Context, charge *stripe.Charge) error {
	partialCharge := &models.PartialCharge{
		ID: charge.ID,
	}

	if charge.Customer != nil {
		partialCharge.CustomerID = &charge.Customer.ID
	}
	if charge.PaymentIntent != nil {
		partialCharge.PaymentIntentID = &charge.PaymentIntent.ID
	}
	if charge.Amount > 0 {
		amount := float64(charge.Amount) / 100
		partialCharge.Amount = &amount
	}
	if charge.Currency != "" {
		chargeCurrency := string(charge.Currency)
		partialCharge.Currency = &chargeCurrency
	}
	if charge.Status != "" {
		chargeStatus := string(charge.Status)
		partialCharge.Status = &chargeStatus
	}
	partialCharge.Paid = &charge.Paid
	partialCharge.Refunded = &charge.Refunded
	if charge.FailureCode != "" {
		partialCharge.FailureCode = &charge.FailureCode
	}
	if charge.FailureMessage != "" {
		partialCharge.FailureMessage = &charge.FailureMessage
	}
	if charge.Created > 0 {
		createdAt := time.Unix(charge.Created, 0)
		partialCharge.CreatedAt = &createdAt
	}

	return sp.charge.Upsert(ctx, partialCharge)
}

func (sp *StripePayment) handleRefundEvent(ctx context.Context, charge *stripe.Charge) error {
	for _, refundData := range charge.Refunds.Data {
		partialRefund := &models.PartialRefund{
			ID: refundData.ID,
		}

		partialRefund.ChargeID = &charge.ID
		if refundData.Amount > 0 {
			amount := float64(refundData.Amount)
			partialRefund.Amount = &amount
		}
		if refundData.Status != "" {
			status := enum.RefundStatus(refundData.Status)
			partialRefund.Status = &status
		}
		if refundData.Reason != "" {
			reason := string(refundData.Reason)
			partialRefund.Reason = &reason
		}
		if refundData.Created > 0 {
			createdAt := time.Unix(refundData.Created, 0)
			partialRefund.CreatedAt = &createdAt
		}

		if err := sp.refundService.Upsert(ctx, partialRefund); err != nil {
			return fmt.Errorf("failed to process refund: %w", err)
		}
	}
	return nil
}

func (sp *StripePayment) handleDisputeEvent(ctx context.Context, dispute *stripe.Dispute, eventType stripe.EventType) error {
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
		disputeStatus := string(dispute.Status)
		partialDispute.Status = &disputeStatus
	}
	if dispute.Reason != "" {
		disputeReason := string(dispute.Reason)
		partialDispute.Reason = &disputeReason
	}
	if dispute.Created > 0 {
		createdAt := time.Unix(dispute.Created, 0)
		partialDispute.CreatedAt = &createdAt
	}

	switch eventType {
	case "charge.dispute.created", "charge.dispute.updated":
		return sp.dispute.Upsert(ctx, partialDispute)
	case "charge.dispute.closed":
		return sp.dispute.Close(ctx, dispute.ID)
	default:
		return fmt.Errorf("unexpected dispute event type: %s", eventType)
	}
}

func (sp *StripePayment) handleProductEvent(ctx context.Context, product *stripe.Product, eventType stripe.EventType) error {
	partialProduct := &models.PartialProduct{
		ID: product.ID,
	}

	if product.Name != "" {
		partialProduct.Name = &product.Name
	}
	if product.Description != "" {
		partialProduct.Description = &product.Description
	}
	partialProduct.Active = &product.Active
	if product.Metadata != nil {
		partialProduct.Metadata = &product.Metadata
	}

	switch eventType {
	case "product.created", "product.updated":
		return sp.productService.Upsert(ctx, partialProduct)
	case "product.deleted":
		return sp.productService.Delete(ctx, product.ID)
	default:
		return fmt.Errorf("unexpected product event type: %s", eventType)
	}
}

func (sp *StripePayment) handlePriceEvent(ctx context.Context, price *stripe.Price, eventType stripe.EventType) error {
	partialPrice := &models.PartialPrice{
		ID: price.ID,
	}

	if price.Product != nil {
		partialPrice.ProductID = &price.Product.ID
	}
	partialPrice.Active = &price.Active
	if price.Currency != "" {
		currency := enum.Currency(price.Currency)
		partialPrice.Currency = &currency
	}
	if price.UnitAmount > 0 {
		unitAmount := float64(price.UnitAmount) / 100
		partialPrice.UnitAmount = &unitAmount
	}
	if price.Type != "" {
		priceType := enum.PriceType(price.Type)
		partialPrice.Type = &priceType
	}
	if price.Recurring != nil {
		if price.Recurring.Interval != "" {
			interval := enum.Interval(price.Recurring.Interval)
			partialPrice.RecurringInterval = &interval
		}
		if price.Recurring.IntervalCount > 0 {
			intervalCount := int32(price.Recurring.IntervalCount)
			partialPrice.RecurringIntervalCount = &intervalCount
		}
	}

	switch eventType {
	case "price.created", "price.updated":
		return sp.priceService.Upsert(ctx, partialPrice)
	case "price.deleted":
		return sp.priceService.Delete(ctx, price.ID)
	default:
		return fmt.Errorf("unexpected price event type: %s", eventType)
	}
}

func (sp *StripePayment) handlePaymentMethodEvent(ctx context.Context, paymentMethod *stripe.PaymentMethod, eventType stripe.EventType) error {
	partialPaymentMethod := &models.PartialPaymentMethod{
		ID: paymentMethod.ID,
	}

	if paymentMethod.Customer != nil {
		partialPaymentMethod.CustomerID = &paymentMethod.Customer.ID
	}
	if paymentMethod.Type != "" {
		pmType := enum.PaymentMethodType(paymentMethod.Type)
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
				cardBrand := string(paymentMethod.Card.Brand)
				partialPaymentMethod.CardBrand = &cardBrand
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

	switch eventType {
	case "payment_method.attached", "payment_method.updated":
		return sp.paymentMethodService.Upsert(ctx, partialPaymentMethod)
	case "payment_method.detached":
		return sp.paymentMethodService.Delete(ctx, paymentMethod.ID)
	default:
		return fmt.Errorf("unexpected payment method event type: %s", eventType)
	}
}

func (sp *StripePayment) handleCouponEvent(ctx context.Context, coupon *stripe.Coupon, eventType stripe.EventType) error {
	partialCoupon := &models.PartialCoupon{
		ID: coupon.ID,
	}

	if coupon.Name != "" {
		partialCoupon.Name = &coupon.Name
	}
	if coupon.Currency != "" {
		currency := string(coupon.Currency)
		partialCoupon.Currency = &currency
	}
	if coupon.Duration != "" {
		duration := string(coupon.Duration)
		partialCoupon.Duration = &duration
	}
	timesRedeemed := int32(coupon.TimesRedeemed)
	partialCoupon.TimesRedeemed = &timesRedeemed
	partialCoupon.Valid = &coupon.Valid
	if coupon.Created > 0 {
		createdAt := time.Unix(coupon.Created, 0)
		partialCoupon.CreatedAt = &createdAt
	}

	if coupon.AmountOff > 0 {
		partialCoupon.AmountOff = &coupon.AmountOff
	}
	if coupon.PercentOff > 0 {
		partialCoupon.PercentOff = &coupon.PercentOff
	}
	if coupon.DurationInMonths > 0 {
		durationInMonths := int(coupon.DurationInMonths)
		partialCoupon.DurationInMonths = &durationInMonths
	}
	if coupon.MaxRedemptions > 0 {
		maxRedemptions := int(coupon.MaxRedemptions)
		partialCoupon.MaxRedemptions = &maxRedemptions
	}
	if coupon.RedeemBy > 0 {
		redeemBy := time.Unix(coupon.RedeemBy, 0)
		partialCoupon.RedeemBy = &redeemBy
	}

	switch eventType {
	case "coupon.created", "coupon.updated":
		return sp.coupon.Upsert(ctx, partialCoupon)
	case "coupon.deleted":
		return sp.coupon.Delete(ctx, coupon.ID)
	default:
		return fmt.Errorf("unexpected coupon event type: %s", eventType)
	}
}

func (sp *StripePayment) handleDiscountEvent(ctx context.Context, discount *stripe.Discount, eventType stripe.EventType) error {
	partialDiscount := &models.PartialDiscount{
		ID: discount.ID,
	}

	if discount.Customer != nil {
		partialDiscount.CustomerID = &discount.Customer.ID
	}
	if discount.Coupon != nil {
		partialDiscount.CouponID = &discount.Coupon.ID
	}
	if discount.Start > 0 {
		start := time.Unix(discount.Start, 0)
		partialDiscount.Start = &start
		partialDiscount.CreatedAt = &start
	}
	if discount.End > 0 {
		end := time.Unix(discount.End, 0)
		partialDiscount.End = &end
	}

	switch eventType {
	case "customer.discount.created", "customer.discount.updated":
		return sp.discount.Upsert(ctx, partialDiscount)
	case "customer.discount.deleted":
		return sp.discount.Delete(ctx, discount.ID)
	default:
		return fmt.Errorf("unexpected discount event type: %s", eventType)
	}
}
