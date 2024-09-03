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

	"goflare.io/payment/models"
	"goflare.io/payment/models/enum"
)

type StripePayment struct {
	client *client.API
}

//func NewStripePaymentService(stripeKey string, db *sql.DB) Payment {
//	sc := &client.API{}
//	sc.Init(stripeKey, nil)
//	return &paymentService{
//		client: sc,
//		db:           db,
//	}
//}

func (s *StripePayment) CreateCustomer(ctx context.Context, userID uint64, email, name string) error {
	params := &stripe.CustomerParams{
		Email: stripe.String(email),
		Name:  stripe.String(name),
	}

	_, err := s.client.Customers.New(params)
	if err != nil {
		return fmt.Errorf("failed to create Stripe customer: %w", err)
	}

	return nil
}

func (s *StripePayment) CreateProduct(ctx context.Context, name, description string, active bool) (*models.Product, error) {

	params := &stripe.ProductParams{
		Name:        stripe.String(name),
		Description: stripe.String(description),
		Active:      stripe.Bool(active),
	}
	stripeProduct, err := s.client.Products.New(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create Stripe product: %w", err)
	}

	product := &models.Product{
		Name:        name,
		Description: description,
		Active:      active,
		StripeID:    stripeProduct.ID,
	}
	// 使用 SQLC 或直接 SQL 插入產品記錄
	// ...

	return product, nil
}

func (s *StripePayment) CreatePrice(ctx context.Context, productID uint64, unitAmount float64, priceType enum.PriceType, currency enum.Currency, interval enum.Interval, intervalCount, trialPeriodDays int32) (*models.Price, error) {
	params := &stripe.PriceParams{
		Product:    stripe.String(strconv.FormatUint(productID, 10)),
		Currency:   stripe.String(string(currency)),
		UnitAmount: stripe.Int64(int64(unitAmount)),
	}
	if priceType == enum.PriceTypeRecurring {
		params.Recurring = &stripe.PriceRecurringParams{
			Interval:      stripe.String(string(interval)),
			IntervalCount: stripe.Int64(int64(intervalCount)),
		}
		if trialPeriodDays > 0 {
			params.Recurring.TrialPeriodDays = stripe.Int64(int64(trialPeriodDays))
		}
	}
	stripePrice, err := s.client.Prices.New(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create Stripe price: %w", err)
	}

	price := &models.Price{
		ProductID:              productID,
		Type:                   priceType,
		Currency:               currency,
		UnitAmount:             unitAmount,
		RecurringInterval:      interval,
		RecurringIntervalCount: intervalCount,
		TrialPeriodDays:        trialPeriodDays,
		StripeID:               stripePrice.ID,
	}
	// 使用 SQLC 或直接 SQL 插入價格記錄
	// ...

	return price, nil
}

func (s *StripePayment) CreateSubscription(ctx context.Context, customerID, priceID uint64) (*models.Subscription, error) {
	params := &stripe.SubscriptionParams{
		Customer: stripe.String(strconv.FormatUint(customerID, 10)),
		Items: []*stripe.SubscriptionItemsParams{
			{
				Price: stripe.String(strconv.FormatUint(priceID, 10)),
			},
		},
	}

	stripeSubscription, err := s.client.Subscriptions.New(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create Stripe subscription: %w", err)
	}

	subscription := &models.Subscription{
		CustomerID:         customerID,
		PriceID:            priceID,
		Status:             enum.SubscriptionStatus(stripeSubscription.Status),
		CurrentPeriodStart: time.Unix(stripeSubscription.CurrentPeriodStart, 0),
		CurrentPeriodEnd:   time.Unix(stripeSubscription.CurrentPeriodEnd, 0),
		StripeID:           stripeSubscription.ID,
	}
	// 使用 SQLC 或直接 SQL 插入訂閱記錄
	// ...

	return subscription, nil
}

func (s *StripePayment) CreatePaymentIntent(ctx context.Context, customerID uint64, amount uint64, currency enum.Currency) (*models.PaymentIntent, error) {
	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(int64(amount)),
		Currency: stripe.String(string(currency)),
		Customer: stripe.String(strconv.FormatUint(customerID, 10)),
	}
	stripePI, err := s.client.PaymentIntents.New(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create Stripe PaymentIntent: %w", err)
	}

	paymentIntent := &models.PaymentIntent{
		CustomerID:   customerID,
		Amount:       amount,
		Currency:     currency,
		Status:       enum.PaymentIntentStatus(stripePI.Status),
		StripeID:     stripePI.ID,
		ClientSecret: stripePI.ClientSecret,
	}
	// 使用 SQLC 或直接 SQL 插入 PaymentIntent 記錄
	// ...

	return paymentIntent, nil
}

// HandleStripeWebhook
//
//	func (s *payment) StripeWebhookHandler(w http.ResponseWriter, r *http.Request) {
//		// 讀取請求體
//		payload, err := io.ReadAll(r.Body)
//		if err != nil {
//			http.Error(w, "Failed to read request body", http.StatusBadRequest)
//			return
//		}
//		defer r.Body.Close()
//
//		// 獲取 Stripe-Signature 標頭
//		signature := r.Header.Get("Stripe-Signature")
//		if signature == "" {
//			http.Error(w, "No Stripe signature found", http.StatusBadRequest)
//			return
//		}
//
//		// 調用 HandleStripeWebhook 函數
//		err = s.HandleStripeWebhook(r.Context(), payload, signature)
//		if err != nil {
//			http.Error(w, "Error processing webhook", http.StatusInternalServerError)
//			return
//		}
//
//		w.WriteHeader(http.StatusOK)
//	}
//
// 即時更新：
//
// 當訂閱狀態改變時（如創建、取消、更新）
// 當發票被支付或失敗時
// 當退款被處理時
//
// 異步支付流程：
//
// 處理需要額外驗證的支付（如 3D Secure）
// 處理非即時支付方式（如銀行轉帳）
//
// 風險管理：
//
// 接收有關潛在欺詐交易的警報
// 處理爭議和退單
//
// 帳戶管理：
//
// 處理客戶資訊更新
// 接收關於帳戶餘額變化的通知
//
// 報告和分析：
//
// 收集實時交易數據
// 觸發自動報告生成
func (s *StripePayment) HandleStripeWebhook(ctx context.Context, payload []byte, signature string) error {
	const webhookSecret = "whsec_..." // 從配置中獲取

	event, err := webhook.ConstructEvent(payload, signature, webhookSecret)
	if err != nil {
		return fmt.Errorf("failed to verify webhook signature: %w", err)
	}

	switch event.Type {
	case "customer.subscription.created", "customer.subscription.updated":
		var subscription stripe.Subscription
		err := json.Unmarshal(event.Data.Raw, &subscription)
		if err != nil {
			return fmt.Errorf("failed to parse subscription data: %w", err)
		}
		// 更新本地訂閱記錄
		// ...
	case "invoice.paid":
		var invoice stripe.Invoice
		err := json.Unmarshal(event.Data.Raw, &invoice)
		if err != nil {
			return fmt.Errorf("failed to parse invoice data: %w", err)
		}
		// 更新本地發票記錄
		// ...
	// 處理其他事件類型...
	default:
		fmt.Printf("Unhandled event type: %s\n", event.Type)
	}

	return nil
}
