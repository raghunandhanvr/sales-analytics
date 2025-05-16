package repository

func NewIngestionRepo(db Database) struct {
	Customers CustomerRepo
	Products  ProductRepo
	Orders    OrderRepo
	Items     ItemRepo
} {
	base := Base{DB: db}
	return struct {
		Customers CustomerRepo
		Products  ProductRepo
		Orders    OrderRepo
		Items     ItemRepo
	}{
		Customers: &customerRepository{Base: base},
		Products:  &productRepository{Base: base},
		Orders:    &orderRepository{Base: base},
		Items:     &itemRepository{Base: base},
	}
}
