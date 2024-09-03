package enum

type PaymentIntentStatus string

const (
	PaymentIntentStatusRequiresPaymentMethod PaymentIntentStatus = "REQUIRES_PAYMENT_METHOD"
	PaymentIntentStatusRequiresConfirmation  PaymentIntentStatus = "REQUIRES_CONFIRMATION"
	PaymentIntentStatusRequiresAction        PaymentIntentStatus = "REQUIRES_ACTION"
	PaymentIntentStatusProcessing            PaymentIntentStatus = "PROCESSING"
	PaymentIntentStatusSucceeded             PaymentIntentStatus = "SUCCEEDED"
	PaymentIntentStatusCanceled              PaymentIntentStatus = "CANCELED"
)
