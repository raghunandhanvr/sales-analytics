package ingestion

import (
	"time"

	"sales-analytics/internal/models"
)

type Sale struct {
	// identifiers
	OrderID    string
	ProductID  string
	CustomerID string

	// order info
	OrderDate  time.Time
	OrderTotal float64

	// product info
	ProductName     string
	ProductCategory string
	Price           float64

	// customer info
	CustomerName    string
	CustomerEmail   string
	CustomerAddress string
	Region          string

	// item details
	Quantity int
	Discount float64
	Shipping float64
}

// ToCustomer converts sale data to a Customer model
func (s Sale) ToCustomer() models.Customer {
	return models.Customer{
		ID:      s.CustomerID,
		Name:    s.CustomerName,
		Email:   s.CustomerEmail,
		Region:  s.Region,
		Address: s.CustomerAddress,
	}
}

// ToProduct converts sale data to a Product model
func (s Sale) ToProduct() models.Product {
	return models.Product{
		ID:        s.ProductID,
		Name:      s.ProductName,
		Category:  s.ProductCategory,
		UnitPrice: s.Price,
	}
}

// EntityType represents different entity types for batching
type EntityType int

const (
	CustomerEntity EntityType = iota
	ProductEntity
	OrderEntity
	OrderItemEntity
)

// Entity is a generic interface for entities that can be batched
type Entity interface {
	Type() EntityType
}
