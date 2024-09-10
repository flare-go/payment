package payment

import (
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/sync/errgroup"
	"strconv"
	"time"

	"go.uber.org/zap"

	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/client"
	"github.com/stripe/stripe-go/v79/webhook"

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
	client          *client.API
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

	dispatcher *Dispatcher
	logger     *zap.Logger
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

	sp.dispatcher = NewDispatcher(10, 100, sp) // 10 workers, 100 job queue size
	sp.dispatcher.Run()

	return sp
}

func (sp *StripePayment) processEvent(ctx context.Context, event *stripe.Event) error {
	// 使用 errgroup 來管理可能的並發操作
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var handleErr error
		start := time.Now()

		switch event.Type {
		// 客戶相關事件
		case "customer.created", "customer.updated", "customer.deleted":
			var customerModel stripe.Customer
			if err := json.Unmarshal(event.Data.Raw, &customerModel); err != nil {
				return fmt.Errorf("解析客戶數據失敗: %w", err)
			}
			handleErr = sp.handleCustomerEvent(ctx, &customerModel, event.Type)

		// 訂閱相關事件
		case "customer.subscription.created", "customer.subscription.updated", "customer.subscription.deleted",
			"customer.subscription.trial_will_end", "customer.subscription.pending_update_applied",
			"customer.subscription.pending_update_expired", "customer.subscription.paused",
			"customer.subscription.resumed":
			var subscriptionModel stripe.Subscription
			if err := json.Unmarshal(event.Data.Raw, &subscriptionModel); err != nil {
				return fmt.Errorf("解析訂閱數據失敗: %w", err)
			}
			handleErr = sp.handleSubscriptionEvent(ctx, &subscriptionModel, event.Type)

		// 發票相關事件
		case "invoice.created", "invoice.updated", "invoice.paid", "invoice.payment_failed",
			"invoice.finalized", "invoice.sent", "invoice.upcoming", "invoice.voided":
			var invoiceModel stripe.Invoice
			if err := json.Unmarshal(event.Data.Raw, &invoiceModel); err != nil {
				return fmt.Errorf("解析發票數據失敗: %w", err)
			}
			handleErr = sp.handleInvoiceEvent(ctx, &invoiceModel, event.Type)

		// 支付意圖相關事件
		case "payment_intent.created", "payment_intent.succeeded", "payment_intent.payment_failed",
			"payment_intent.canceled", "payment_intent.processing", "payment_intent.requires_action":
			var paymentIntentModel stripe.PaymentIntent
			if err := json.Unmarshal(event.Data.Raw, &paymentIntentModel); err != nil {
				return fmt.Errorf("解析支付意圖數據失敗: %w", err)
			}
			handleErr = sp.handlePaymentIntentEvent(ctx, &paymentIntentModel)

		// 收費相關事件
		case "charge.succeeded", "charge.failed", "charge.refunded", "charge.captured", "charge.expired":
			var chargeModel stripe.Charge
			if err := json.Unmarshal(event.Data.Raw, &chargeModel); err != nil {
				return fmt.Errorf("解析收費數據失敗: %w", err)
			}
			handleErr = sp.handleChargeEvent(ctx, &chargeModel)

		// 爭議相關事件
		case "charge.dispute.created", "charge.dispute.updated", "charge.dispute.closed",
			"charge.dispute.funds_reinstated", "charge.dispute.funds_withdrawn":
			var disputeModel stripe.Dispute
			if err := json.Unmarshal(event.Data.Raw, &disputeModel); err != nil {
				return fmt.Errorf("解析爭議數據失敗: %w", err)
			}
			handleErr = sp.handleDisputeEvent(ctx, &disputeModel, event.Type)

		// 產品相關事件
		case "product.created", "product.updated", "product.deleted":
			var productModel stripe.Product
			if err := json.Unmarshal(event.Data.Raw, &productModel); err != nil {
				return fmt.Errorf("解析產品數據失敗: %w", err)
			}
			handleErr = sp.handleProductEvent(ctx, &productModel, event.Type)

		// 價格相關事件
		case "price.created", "price.updated", "price.deleted":
			var priceModel stripe.Price
			if err := json.Unmarshal(event.Data.Raw, &priceModel); err != nil {
				return fmt.Errorf("解析價格數據失敗: %w", err)
			}
			handleErr = sp.handlePriceEvent(ctx, &priceModel, event.Type)

		// 支付方式相關事件
		case "payment_method.attached", "payment_method.updated", "payment_method.detached":
			var paymentMethodModel stripe.PaymentMethod
			if err := json.Unmarshal(event.Data.Raw, &paymentMethodModel); err != nil {
				return fmt.Errorf("解析支付方式數據失敗: %w", err)
			}
			handleErr = sp.handlePaymentMethodEvent(ctx, &paymentMethodModel, event.Type)

		// 優惠券相關事件
		case "coupon.created", "coupon.updated", "coupon.deleted":
			var couponModel stripe.Coupon
			if err := json.Unmarshal(event.Data.Raw, &couponModel); err != nil {
				return fmt.Errorf("解析優惠券數據失敗: %w", err)
			}
			handleErr = sp.handleCouponEvent(ctx, &couponModel, event.Type)

		// 折扣相關事件
		case "customer.discount.created", "customer.discount.updated", "customer.discount.deleted":
			var discountModel stripe.Discount
			if err := json.Unmarshal(event.Data.Raw, &discountModel); err != nil {
				return fmt.Errorf("解析折扣數據失敗: %w", err)
			}
			handleErr = sp.handleDiscountEvent(ctx, &discountModel, event.Type)

		// 結帳會話相關事件
		case "checkout.session.completed", "checkout.session.async_payment_succeeded",
			"checkout.session.async_payment_failed", "checkout.session.expired":
			var checkoutSessionModel stripe.CheckoutSession
			if err := json.Unmarshal(event.Data.Raw, &checkoutSessionModel); err != nil {
				return fmt.Errorf("解析結帳會話數據失敗: %w", err)
			}
			handleErr = sp.handleCheckoutSessionEvent(ctx, &checkoutSessionModel, event.Type)

		// 退款相關事件
		case "refund.created", "refund.updated":
			var refundModel stripe.Refund
			if err := json.Unmarshal(event.Data.Raw, &refundModel); err != nil {
				return fmt.Errorf("解析退款數據失敗: %w", err)
			}
			handleErr = sp.handleRefundEvent(ctx, &refundModel)

		// 促銷代碼相關事件
		case "promotion_code.created", "promotion_code.updated":
			var promotionCodeModel stripe.PromotionCode
			if err := json.Unmarshal(event.Data.Raw, &promotionCodeModel); err != nil {
				return fmt.Errorf("解析促銷代碼數據失敗: %w", err)
			}
			handleErr = sp.handlePromotionCodeEvent(ctx, &promotionCodeModel, event.Type)

		// 報價相關事件
		case "quote.created", "quote.finalized", "quote.accepted", "quote.canceled":
			var quoteModel stripe.Quote
			if err := json.Unmarshal(event.Data.Raw, &quoteModel); err != nil {
				return fmt.Errorf("解析報價數據失敗: %w", err)
			}
			handleErr = sp.handleQuoteEvent(ctx, &quoteModel, event.Type)

		// 支付連結相關事件
		case "payment_link.created", "payment_link.updated":
			var paymentLinkModel stripe.PaymentLink
			if err := json.Unmarshal(event.Data.Raw, &paymentLinkModel); err != nil {
				return fmt.Errorf("解析支付連結數據失敗: %w", err)
			}
			handleErr = sp.handlePaymentLinkEvent(ctx, &paymentLinkModel, event.Type)

		// 稅率相關事件
		case "tax_rate.sql.created", "tax_rate.sql.updated":
			var taxRateModel stripe.TaxRate
			if err := json.Unmarshal(event.Data.Raw, &taxRateModel); err != nil {
				return fmt.Errorf("解析稅率數據失敗: %w", err)
			}
			handleErr = sp.handleTaxRateEvent(ctx, &taxRateModel, event.Type)

		// 審查相關事件
		case "review.opened", "review.closed":
			var reviewModel stripe.Review
			if err := json.Unmarshal(event.Data.Raw, &reviewModel); err != nil {
				return fmt.Errorf("解析審查數據失敗: %w", err)
			}
			handleErr = sp.handleReviewEvent(ctx, &reviewModel, event.Type)

		default:
			sp.logger.Info("未處理的事件類型",
				zap.String("type", string(event.Type)),
				zap.String("event_id", event.ID))
			return nil
		}

		if handleErr != nil {
			return fmt.Errorf("處理事件 %s 時發生錯誤: %w", event.Type, handleErr)
		}

		duration := time.Since(start)
		sp.logger.Info("事件處理成功",
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
		return fmt.Errorf("標記事件為已處理失敗: %w", err)
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
func (sp *StripePayment) UpdateCustomerBalance(ctx context.Context, updateCustomer *models.Customer) error {

	params := &stripe.CustomerParams{
		Balance: &updateCustomer.Balance,
	}

	if _, err := sp.client.Customers.Update(updateCustomer.ID, params); err != nil {
		return fmt.Errorf("failed to update Stripe customer: %w", err)
	}

	if err := sp.customer.UpdateBalance(ctx, updateCustomer.ID, uint64(updateCustomer.Balance)); err != nil {
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
	stripeEvent, err := webhook.ConstructEvent(payload, signature, "")
	if err != nil {
		return fmt.Errorf("failed to verify webhook signature: %w", err)
	}

	processed, err := sp.event.IsEventProcessed(ctx, stripeEvent.ID)
	if err != nil {
		return fmt.Errorf("failed to check event status: %w", err)
	}
	if processed {
		return nil
	}

	sp.dispatcher.jobQueue <- WorkRequest{
		Event: &stripeEvent,
		Ctx:   ctx,
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
		return sp.customer.Upsert(ctx, partialCustomer)
	case "customer.deleted":
		return sp.customer.Delete(ctx, customer.ID)
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
		partialSubscription.Status = &subscription.Status
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
		return sp.subscription.Upsert(ctx, partialSubscription)
	case "customer.subscription.deleted":
		return sp.subscription.Delete(ctx, subscription.ID)
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
		partialInvoice.Status = &invoice.Status
	}
	if invoice.Currency != "" {
		partialInvoice.Currency = &invoice.Currency
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
		return sp.invoice.Upsert(ctx, partialInvoice)
	case "invoice.deleted":
		return sp.invoice.Delete(ctx, invoice.ID)
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

	return sp.paymentIntent.Upsert(ctx, partialPaymentIntent)
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
		partialCharge.Currency = &charge.Currency
	}
	if charge.Status != "" {
		partialCharge.Status = &charge.Status
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

func (sp *StripePayment) handleRefundEvent(ctx context.Context, refund *stripe.Refund) error {
	partialRefund := &models.PartialRefund{
		ID: refund.ID,
	}

	if refund.Charge != nil {
		partialRefund.ChargeID = &refund.Charge.ID
	}
	if refund.Amount > 0 {
		amount := float64(refund.Amount)
		partialRefund.Amount = &amount
	}
	if refund.Status != "" {
		partialRefund.Status = &refund.Status
	}
	if refund.Reason != "" {
		partialRefund.Reason = &refund.Reason
	}
	if refund.Created > 0 {
		createdAt := time.Unix(refund.Created, 0)
		partialRefund.CreatedAt = &createdAt
	}

	if err := sp.refund.Upsert(ctx, partialRefund); err != nil {
		return fmt.Errorf("處理退款失敗: %w", err)
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
		partialDispute.Status = &dispute.Status
	}
	if dispute.Reason != "" {
		partialDispute.Reason = &dispute.Reason
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
		return sp.product.Upsert(ctx, partialProduct)
	case "product.deleted":
		return sp.product.Delete(ctx, product.ID)
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
		partialPrice.Currency = &price.Currency
	}
	if price.UnitAmount > 0 {
		unitAmount := float64(price.UnitAmount) / 100
		partialPrice.UnitAmount = &unitAmount
	}
	if price.Type != "" {
		partialPrice.Type = &price.Type
	}
	if price.Recurring != nil {
		if price.Recurring.Interval != "" {
			partialPrice.RecurringInterval = &price.Recurring.Interval
		}
		if price.Recurring.IntervalCount > 0 {
			intervalCount := int32(price.Recurring.IntervalCount)
			partialPrice.RecurringIntervalCount = &intervalCount
		}
	}

	switch eventType {
	case "price.created", "price.updated":
		return sp.price.Upsert(ctx, partialPrice)
	case "price.deleted":
		return sp.price.Delete(ctx, price.ID)
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

	switch eventType {
	case "payment_method.attached", "payment_method.updated":
		return sp.paymentMethod.Upsert(ctx, partialPaymentMethod)
	case "payment_method.detached":
		return sp.paymentMethod.Delete(ctx, paymentMethod.ID)
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
		partialCoupon.Currency = &coupon.Currency
	}
	if coupon.Duration != "" {
		partialCoupon.Duration = &coupon.Duration
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

func (sp *StripePayment) handlePromotionCodeEvent(ctx context.Context, promotionCode *stripe.PromotionCode, eventType stripe.EventType) error {
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

	switch eventType {
	case "promotion_code.created", "promotion_code.updated":
		return sp.promotionCode.Upsert(ctx, partialPromotionCode)
	default:
		return fmt.Errorf("unexpected promotion code event type: %s", eventType)
	}
}

func (sp *StripePayment) handleCheckoutSessionEvent(ctx context.Context, session *stripe.CheckoutSession, eventType stripe.EventType) error {
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

	switch eventType {
	case "checkout.session.completed", "checkout.session.async_payment_succeeded",
		"checkout.session.async_payment_failed", "checkout.session.expired":
		return sp.checkoutSession.Upsert(ctx, partialSession)
	default:
		return fmt.Errorf("unexpected checkout session event type: %s", eventType)
	}
}

func (sp *StripePayment) handleQuoteEvent(ctx context.Context, quote *stripe.Quote, eventType stripe.EventType) error {
	partialQuote := &models.PartialQuote{
		ID: quote.ID,
	}

	if quote.Customer != nil {
		partialQuote.CustomerID = &quote.Customer.ID
	}
	partialQuote.Status = &quote.Status

	amountTotal := quote.AmountTotal
	partialQuote.AmountTotal = &amountTotal

	partialQuote.Currency = &quote.Currency

	if quote.ExpiresAt > 0 {
		validUntil := time.Unix(quote.ExpiresAt, 0)
		partialQuote.ValidUntil = &validUntil
	}

	// Stripe的Quote模型中沒有直接的AcceptedAt字段，
	// 從StatusTransitions中獲取，如果存在的話
	if quote.StatusTransitions != nil && quote.StatusTransitions.AcceptedAt > 0 {
		acceptedAt := time.Unix(quote.StatusTransitions.AcceptedAt, 0)
		partialQuote.AcceptedAt = &acceptedAt
	}

	// CanceledAt也可以從StatusTransitions中獲取
	if quote.StatusTransitions != nil && quote.StatusTransitions.CanceledAt > 0 {
		canceledAt := time.Unix(quote.StatusTransitions.CanceledAt, 0)
		partialQuote.CanceledAt = &canceledAt
	}

	if quote.Created > 0 {
		createdAt := time.Unix(quote.Created, 0)
		partialQuote.CreatedAt = &createdAt
	}

	switch eventType {
	case "quote.created", "quote.finalized", "quote.accepted", "quote.canceled":
		return sp.quote.Upsert(ctx, partialQuote)
	default:
		return fmt.Errorf("unexpected quote event type: %s", eventType)
	}
}

func (sp *StripePayment) handlePaymentLinkEvent(ctx context.Context, paymentLink *stripe.PaymentLink, eventType stripe.EventType) error {
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

	switch eventType {
	case "payment_link.created", "payment_link.updated":
		return sp.paymentLink.Upsert(ctx, partialPaymentLink)
	default:
		return fmt.Errorf("unexpected payment link event type: %s", eventType)
	}
}

func (sp *StripePayment) handleTaxRateEvent(ctx context.Context, taxRate *stripe.TaxRate, eventType stripe.EventType) error {
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

	switch eventType {
	case "tax_rate.created", "tax_rate.updated":
		return sp.taxRate.Upsert(ctx, partialTaxRate)
	default:
		return fmt.Errorf("unexpected tax rate event type: %s", eventType)
	}
}

func (sp *StripePayment) handleReviewEvent(ctx context.Context, review *stripe.Review, eventType stripe.EventType) error {
	partialReview := &models.PartialReview{
		ID: review.ID,
	}

	if review.PaymentIntent != nil {
		partialReview.PaymentIntentID = &review.PaymentIntent.ID
	}
	partialReview.Reason = &review.Reason

	// 根據 Open 字段設置狀態
	var status string
	if review.Open {
		status = "open"
	} else {
		status = "closed"
	}
	partialReview.Status = &status

	if review.Created > 0 {
		createdAt := time.Unix(review.Created, 0)
		partialReview.CreatedAt = &createdAt
		// 使用 Created 時間作為 OpenedAt
		partialReview.OpenedAt = &createdAt
	}

	// 如果 Review 已關閉，設置 ClosedAt
	if !review.Open {
		closedAt := time.Now()
		partialReview.ClosedAt = &closedAt
	}

	// ClosedReason
	if review.ClosedReason != "" {
		partialReview.ClosedReason = &review.ClosedReason
	}

	switch eventType {
	case "review.opened", "review.closed":
		return sp.review.Upsert(ctx, partialReview)
	default:
		return fmt.Errorf("unexpected review event type: %s", eventType)
	}
}

func (sp *StripePayment) Close() {
	sp.logger.Info("Initiating graceful shutdown of workers and dispatcher")
	sp.dispatcher.Stop() // 停止 dispatcher 和 workers
	sp.logger.Info("StripePayment successfully shutdown")
}
