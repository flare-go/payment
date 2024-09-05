package models

import (
	"time"

	"goflare.io/payment/models/enum"
	"goflare.io/payment/sqlc"
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

func NewPaymentMethod() *PaymentMethod {
	return &PaymentMethod{}
}

func (pm *PaymentMethod) ConvertFromSQLCPaymentMethod(sqlcPaymentMethod any) *PaymentMethod {

	var (
		id, customerID                                                        uint64
		paymentMethodType                                                     enum.PaymentMethodType
		cardLast4, cardBrand, bankAccountLast4, bankAccountBankName, stripeID string
		cardExpMonth, cardExpYear                                             int32
		isDefault                                                             bool
		createdAt, updatedAt                                                  time.Time
	)

	switch sp := sqlcPaymentMethod.(type) {
	case *sqlc.PaymentMethod:
		id = sp.ID
		customerID = sp.CustomerID
		paymentMethodType = enum.PaymentMethodType(sp.Type)
		if sp.CardLast4 != nil {
			cardLast4 = *sp.CardLast4
		}
		if sp.CardBrand != nil {
			cardBrand = *sp.CardBrand
		}
		if sp.CardExpMonth != nil {
			cardExpMonth = *sp.CardExpMonth
		}
		if sp.CardExpYear != nil {
			cardExpYear = *sp.CardExpYear
		}
		if sp.BankAccountLast4 != nil {
			bankAccountLast4 = *sp.BankAccountLast4
		}
		if sp.BankAccountBankName != nil {
			bankAccountBankName = *sp.BankAccountBankName
		}
		stripeID = sp.StripeID
		isDefault = sp.IsDefault
		createdAt = sp.CreatedAt.Time
		updatedAt = sp.UpdatedAt.Time
	default:
		return nil
	}

	pm.ID = id
	pm.CustomerID = customerID
	pm.Type = paymentMethodType
	pm.CardLast4 = cardLast4
	pm.CardBrand = cardBrand
	pm.CardExpMonth = cardExpMonth
	pm.CardExpYear = cardExpYear
	pm.BankAccountLast4 = bankAccountLast4
	pm.BankAccountBankName = bankAccountBankName
	pm.StripeID = stripeID
	pm.IsDefault = isDefault
	pm.CreatedAt = createdAt
	pm.UpdatedAt = updatedAt

	return pm
}
