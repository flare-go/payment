package enum

type PriceType string

const (
	PriceTypeOneTime   PriceType = "ONE_TIME"
	PriceTypeRecurring PriceType = "RECURRING"
)
