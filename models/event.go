package models

import "time"

type Event struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Processed bool      `json:"processed"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
