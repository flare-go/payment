package enum

type RefundStatus string

const (
	RefundStatusPending   RefundStatus = "PENDING"
	RefundStatusSucceeded RefundStatus = "SUCCEEDED"
	RefundStatusFailed    RefundStatus = "FAILED"
	RefundStatusCanceled  RefundStatus = "CANCELED"
)
