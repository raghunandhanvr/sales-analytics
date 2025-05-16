package models

type OrderItem struct {
	OrderID, ProductID                string
	Quantity                          int
	UnitPrice, Discount, ShippingCost float64
}
