package models

import "time"

type TaxRate struct {
	ID           string    `json:"id"`
	DisplayName  string    `json:"display_name"`
	Description  string    `json:"description"`
	Jurisdiction string    `json:"jurisdiction"`
	Percentage   float64   `json:"percentage"`
	Inclusive    bool      `json:"inclusive"`
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type PartialTaxRate struct {
	ID           string     `json:"id"`
	DisplayName  *string    `json:"display_name,omitempty"`
	Description  *string    `json:"description,omitempty"`
	Jurisdiction *string    `json:"jurisdiction,omitempty"`
	Percentage   *float64   `json:"percentage,omitempty"`
	Inclusive    *bool      `json:"inclusive,omitempty"`
	Active       *bool      `json:"active,omitempty"`
	CreatedAt    *time.Time `json:"created_at,omitempty"`
	UpdatedAt    *time.Time `json:"updated_at,omitempty"`
}
