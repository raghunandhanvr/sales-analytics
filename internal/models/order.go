package models

import "time"

type Order struct {
	ID, CustomerID string
	OrderDate      time.Time
	TotalAmount    float64
}
