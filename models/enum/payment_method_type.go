package enum

type PaymentMethodType string

const (
	PaymentMethodTypeCard PaymentMethodType = "CARD"
	PaymentMethodTypeBank PaymentMethodType = "BANK"
)
