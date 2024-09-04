package enum

type InvoiceStatus string

const (
	InvoiceStatusDraft         InvoiceStatus = "DRAFT"
	InvoiceStatusOpen          InvoiceStatus = "OPEN"
	InvoiceStatusPaid          InvoiceStatus = "PAID"
	InvoiceStatusPartiallyPaid InvoiceStatus = "PARTIALLY_PAID"
	InvoiceStatusUncollectible InvoiceStatus = "UNCOLLECTIBLE"
	InvoiceStatusVoid          InvoiceStatus = "VOID"
	InvoiceStatusPaymentFailed InvoiceStatus = "PAYMENT_FAILED"
)
