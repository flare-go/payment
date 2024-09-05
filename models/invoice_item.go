package models

import (
	"goflare.io/payment/sqlc"
)

type InvoiceItem struct {
	ID          uint64  `json:"id"`
	InvoiceID   uint64  `json:"invoice_id"`
	Amount      float64 `json:"amount"`
	Description string  `json:"description"`
}

func NewInvoiceItem() *InvoiceItem {
	return &InvoiceItem{}
}

func (item *InvoiceItem) ConvertFromSQLCInvoiceItem(sqlcInvoiceItem any) *InvoiceItem {

	var id, invoiceID uint64
	var amount float64
	var desc string

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
