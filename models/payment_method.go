package models

import (
	"time"

	"github.com/stripe/stripe-go/v79"

	"goflare.io/payment/sqlc"
)

// PaymentMethod 代表客戶的支付方式
// PaymentMethod represents a customer's payment method
type PaymentMethod struct {
	ID                  string
	CustomerID          string
	Type                stripe.PaymentMethodType
	CardLast4           string
	CardBrand           stripe.PaymentMethodCardBrand
	CardExpMonth        int32
	CardExpYear         int32
	BankAccountLast4    string
	BankAccountBankName string
	IsDefault           bool
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type PartialPaymentMethod struct {
	ID                  string
	CustomerID          *string
	Type                *stripe.PaymentMethodType
	CardLast4           *string
	CardBrand           *stripe.PaymentMethodCardBrand
	CardExpMonth        *int32
	CardExpYear         *int32
	BankAccountLast4    *string
	BankAccountBankName *string
	CreatedAt           *time.Time
	UpdatedAt           *time.Time
}

func NewPaymentMethod() *PaymentMethod {
	return &PaymentMethod{}
}

func (pm *PaymentMethod) ConvertFromSQLCPaymentMethod(sqlcPaymentMethod any) *PaymentMethod {

	var (
		paymentMethodType                                                stripe.PaymentMethodType
		cardBrand                                                        stripe.PaymentMethodCardBrand
		id, customerID, cardLast4, bankAccountLast4, bankAccountBankName string
		cardExpMonth, cardExpYear                                        int32
		isDefault                                                        bool
		createdAt, updatedAt                                             time.Time
	)

	switch sp := sqlcPaymentMethod.(type) {
	case *sqlc.PaymentMethod:
		id = sp.ID
		customerID = sp.CustomerID
		paymentMethodType = stripe.PaymentMethodType(sp.Type)
		if sp.CardLast4 != nil {
			cardLast4 = *sp.CardLast4
		}
		if sp.CardBrand.Valid {
			cardBrand = stripe.PaymentMethodCardBrand(sp.CardBrand.PaymentMethodCardBrand)
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
	pm.IsDefault = isDefault
	pm.CreatedAt = createdAt
	pm.UpdatedAt = updatedAt

	return pm
}
