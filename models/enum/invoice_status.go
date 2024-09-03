package enum

type InvoiceStatus string

const (
	InvoiceStatusDraft         InvoiceStatus = "DRAFT"
	InvoiceStatusOpen          InvoiceStatus = "OPEN"
	InvoiceStatusPaid          InvoiceStatus = "PAID"
	InvoiceStatusUncollectible InvoiceStatus = "UNCOLLECTIBLE"
	InvoiceStatusVoid          InvoiceStatus = "VOID"
)
