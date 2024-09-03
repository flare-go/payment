package models

import (
	"goflare.io/payment/models/enum"
	"time"
)

// PaymentMethod 代表客戶的支付方式
// PaymentMethod represents a customer's payment method
type PaymentMethod struct {
	ID                  uint64
	CustomerID          uint64
	Type                enum.PaymentMethodType
	CardLast4           string
	CardBrand           string
	CardExpMonth        int32
	CardExpYear         int32
	BankAccountLast4    string // 添加銀行帳號後四位
	BankAccountBankName string // 添加銀行名稱
	IsDefault           bool   // 添加是否為默認支付方式
	StripeID            string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}
