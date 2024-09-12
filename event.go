package payment

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/stripe/stripe-go/v79"
	"go.uber.org/zap"
)

type EventHandler func(context.Context, *stripe.Event) error

type EventManager struct {
	natsConn *nats.Conn
	handlers map[stripe.EventType]EventHandler
	logger   *zap.Logger
}

func NewEventManager(natsConn *nats.Conn, logger *zap.Logger) *EventManager {
	return &EventManager{
		natsConn: natsConn,
		handlers: make(map[stripe.EventType]EventHandler),
		logger:   logger,
	}
}

func (em *EventManager) RegisterHandler(eventType stripe.EventType, handler EventHandler) {
	em.handlers[eventType] = handler
}

func (em *EventManager) GetHandler(eventType stripe.EventType) (EventHandler, bool) {
	handler, exists := em.handlers[eventType]
	return handler, exists
}

func (em *EventManager) PublishEvent(event *stripe.Event) error {
	subject := fmt.Sprintf("stripe.event.%s", event.Type)
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	return em.natsConn.Publish(subject, data)
}

func (em *EventManager) SubscribeToEvents(wp *WorkerPool) error {
	_, err := em.natsConn.Subscribe("stripe.event.>", func(msg *nats.Msg) {
		var event stripe.Event
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			em.logger.Error("Failed to unmarshal event", zap.Error(err))
			return
		}

		wp.Submit(context.Background(), &event)
	})

	return err
}

func (sp *StripePayment) registerEventHandlers() {

	eventHandlers := map[stripe.EventType]EventHandler{
		// Customer
		stripe.EventTypeCustomerCreated: sp.handleCustomerEvent,
		stripe.EventTypeCustomerUpdated: sp.handleCustomerEvent,
		stripe.EventTypeCustomerDeleted: sp.handleCustomerEvent,

		// Subscription
		stripe.EventTypeSubscriptionScheduleAborted:              sp.handleSubscriptionEvent,
		stripe.EventTypeSubscriptionScheduleCanceled:             sp.handleSubscriptionEvent,
		stripe.EventTypeSubscriptionScheduleCompleted:            sp.handleSubscriptionEvent,
		stripe.EventTypeSubscriptionScheduleCreated:              sp.handleSubscriptionEvent,
		stripe.EventTypeSubscriptionScheduleExpiring:             sp.handleSubscriptionEvent,
		stripe.EventTypeSubscriptionScheduleReleased:             sp.handleSubscriptionEvent,
		stripe.EventTypeSubscriptionScheduleUpdated:              sp.handleSubscriptionEvent,
		stripe.EventTypeCustomerSubscriptionCreated:              sp.handleSubscriptionEvent,
		stripe.EventTypeCustomerSubscriptionDeleted:              sp.handleSubscriptionEvent,
		stripe.EventTypeCustomerSubscriptionPaused:               sp.handleSubscriptionEvent,
		stripe.EventTypeCustomerSubscriptionPendingUpdateApplied: sp.handleSubscriptionEvent,
		stripe.EventTypeCustomerSubscriptionPendingUpdateExpired: sp.handleSubscriptionEvent,
		stripe.EventTypeCustomerSubscriptionResumed:              sp.handleSubscriptionEvent,
		stripe.EventTypeCustomerSubscriptionTrialWillEnd:         sp.handleSubscriptionEvent,
		stripe.EventTypeCustomerSubscriptionUpdated:              sp.handleSubscriptionEvent,

		// Invoice
		stripe.EventTypeInvoiceCreated:               sp.handleInvoiceEvent,
		stripe.EventTypeInvoiceDeleted:               sp.handleInvoiceEvent,
		stripe.EventTypeInvoiceFinalizationFailed:    sp.handleInvoiceEvent,
		stripe.EventTypeInvoiceFinalized:             sp.handleInvoiceEvent,
		stripe.EventTypeInvoiceMarkedUncollectible:   sp.handleInvoiceEvent,
		stripe.EventTypeInvoiceOverdue:               sp.handleInvoiceEvent,
		stripe.EventTypeInvoicePaid:                  sp.handleInvoiceEvent,
		stripe.EventTypeInvoicePaymentActionRequired: sp.handleInvoiceEvent,
		stripe.EventTypeInvoicePaymentFailed:         sp.handleInvoiceEvent,
		stripe.EventTypeInvoicePaymentSucceeded:      sp.handleInvoiceEvent,
		stripe.EventTypeInvoiceSent:                  sp.handleInvoiceEvent,
		stripe.EventTypeInvoiceUpcoming:              sp.handleInvoiceEvent,
		stripe.EventTypeInvoiceUpdated:               sp.handleInvoiceEvent,
		stripe.EventTypeInvoiceVoided:                sp.handleInvoiceEvent,
		stripe.EventTypeInvoiceWillBeDue:             sp.handleInvoiceEvent,
		stripe.EventTypeInvoiceItemCreated:           sp.handleInvoiceEvent,
		stripe.EventTypeInvoiceItemDeleted:           sp.handleInvoiceEvent,

		// Payment Intent
		stripe.EventTypePaymentIntentAmountCapturableUpdated: sp.handlePaymentIntentEvent,
		stripe.EventTypePaymentIntentCanceled:                sp.handlePaymentIntentEvent,
		stripe.EventTypePaymentIntentCreated:                 sp.handlePaymentIntentEvent,
		stripe.EventTypePaymentIntentPartiallyFunded:         sp.handlePaymentIntentEvent,
		stripe.EventTypePaymentIntentPaymentFailed:           sp.handlePaymentIntentEvent,
		stripe.EventTypePaymentIntentProcessing:              sp.handlePaymentIntentEvent,
		stripe.EventTypePaymentIntentRequiresAction:          sp.handlePaymentIntentEvent,
		stripe.EventTypePaymentIntentSucceeded:               sp.handlePaymentIntentEvent,

		// Charge
		stripe.EventTypeChargeCaptured:      sp.handleChargeEvent,
		stripe.EventTypeChargeExpired:       sp.handleChargeEvent,
		stripe.EventTypeChargeFailed:        sp.handleChargeEvent,
		stripe.EventTypeChargePending:       sp.handleChargeEvent,
		stripe.EventTypeChargeRefundUpdated: sp.handleChargeEvent,
		stripe.EventTypeChargeRefunded:      sp.handleChargeEvent,
		stripe.EventTypeChargeSucceeded:     sp.handleChargeEvent,
		stripe.EventTypeChargeUpdated:       sp.handleChargeEvent,

		// Dispute
		stripe.EventTypeChargeDisputeClosed:           sp.handleDisputeEvent,
		stripe.EventTypeChargeDisputeCreated:          sp.handleDisputeEvent,
		stripe.EventTypeChargeDisputeFundsReinstated:  sp.handleDisputeEvent,
		stripe.EventTypeChargeDisputeFundsWithdrawn:   sp.handleDisputeEvent,
		stripe.EventTypeChargeDisputeUpdated:          sp.handleDisputeEvent,
		stripe.EventTypeIssuingDisputeClosed:          sp.handleDisputeEvent,
		stripe.EventTypeIssuingDisputeCreated:         sp.handleDisputeEvent,
		stripe.EventTypeIssuingDisputeFundsReinstated: sp.handleDisputeEvent,
		stripe.EventTypeIssuingDisputeFundsRescinded:  sp.handleDisputeEvent,
		stripe.EventTypeIssuingDisputeSubmitted:       sp.handleDisputeEvent,
		stripe.EventTypeIssuingDisputeUpdated:         sp.handleDisputeEvent,

		// Product
		stripe.EventTypeProductCreated: sp.handleProductEvent,
		stripe.EventTypeProductDeleted: sp.handleProductEvent,
		stripe.EventTypeProductUpdated: sp.handleProductEvent,

		// Price
		stripe.EventTypePriceCreated: sp.handlePriceEvent,
		stripe.EventTypePriceDeleted: sp.handlePriceEvent,
		stripe.EventTypePriceUpdated: sp.handlePriceEvent,

		// Payment Method
		stripe.EventTypePaymentMethodAttached:             sp.handlePaymentMethodEvent,
		stripe.EventTypePaymentMethodAutomaticallyUpdated: sp.handlePaymentMethodEvent,
		stripe.EventTypePaymentMethodDetached:             sp.handlePaymentMethodEvent,
		stripe.EventTypePaymentMethodUpdated:              sp.handlePaymentMethodEvent,

		// Coupon
		stripe.EventTypeCouponCreated: sp.handleCouponEvent,
		stripe.EventTypeCouponDeleted: sp.handleCouponEvent,
		stripe.EventTypeCouponUpdated: sp.handleCouponEvent,

		// Discount
		stripe.EventTypeCustomerDiscountCreated: sp.handleDiscountEvent,
		stripe.EventTypeCustomerDiscountDeleted: sp.handleDiscountEvent,
		stripe.EventTypeCustomerDiscountUpdated: sp.handleDiscountEvent,

		// Checkout Session
		stripe.EventTypeCheckoutSessionAsyncPaymentFailed:    sp.handleCheckoutSessionEvent,
		stripe.EventTypeCheckoutSessionAsyncPaymentSucceeded: sp.handleCheckoutSessionEvent,
		stripe.EventTypeCheckoutSessionCompleted:             sp.handleCheckoutSessionEvent,
		stripe.EventTypeCheckoutSessionExpired:               sp.handleCheckoutSessionEvent,

		// Refund
		stripe.EventTypeRefundCreated: sp.handleRefundEvent,
		stripe.EventTypeRefundUpdated: sp.handleRefundEvent,

		// Promotion Code
		stripe.EventTypePromotionCodeCreated: sp.handlePromotionCodeEvent,
		stripe.EventTypePromotionCodeUpdated: sp.handlePromotionCodeEvent,

		// Quote
		stripe.EventTypeQuoteAccepted:  sp.handleQuoteEvent,
		stripe.EventTypeQuoteCanceled:  sp.handleQuoteEvent,
		stripe.EventTypeQuoteCreated:   sp.handleQuoteEvent,
		stripe.EventTypeQuoteFinalized: sp.handleQuoteEvent,

		// Payment Link
		stripe.EventTypePaymentLinkCreated: sp.handlePaymentLinkEvent,
		stripe.EventTypePaymentLinkUpdated: sp.handlePaymentLinkEvent,

		// Tax Rate
		stripe.EventTypeTaxRateCreated: sp.handleTaxRateEvent,
		stripe.EventTypeTaxRateUpdated: sp.handleTaxRateEvent,

		// Review
		stripe.EventTypeReviewClosed: sp.handleReviewEvent,
		stripe.EventTypeReviewOpened: sp.handleReviewEvent,
	}

	// 使用 map 來註冊所有的事件處理器
	for eventType, handler := range eventHandlers {
		sp.eventManager.RegisterHandler(eventType, handler)
	}
}
