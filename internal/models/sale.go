package models

type Sale struct {
	Customer  Customer
	Product   Product
	Order     Order
	OrderItem OrderItem
}
