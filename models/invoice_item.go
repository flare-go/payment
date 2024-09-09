package models

import (
	"goflare.io/payment/sqlc"
)

type InvoiceItem struct {
	ID          string  `json:"id"`
	InvoiceID   string  `json:"invoice_id"`
	Amount      float64 `json:"amount"`
	Description string  `json:"description"`
}

func NewInvoiceItem() *InvoiceItem {
	return &InvoiceItem{}
}

func (item *InvoiceItem) ConvertFromSQLCInvoiceItem(sqlcInvoiceItem any) *InvoiceItem {

	var id, invoiceID, desc string
	var amount float64

	switch sp := sqlcInvoiceItem.(type) {
	case *sqlc.InvoiceItem:
		id = sp.ID
		invoiceID = sp.InvoiceID
		amount = sp.Amount
		if sp.Description != nil {
			desc = *sp.Description
		}
	default:
		return nil
	}

	item.ID = id
	item.InvoiceID = invoiceID
	item.Amount = amount
	item.Description = desc

	return item
}
