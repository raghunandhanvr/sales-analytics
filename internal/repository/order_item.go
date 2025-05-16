package repository

import (
	"context"
)

type itemRepository struct {
	Base
}

func NewItemRepo(db Database) ItemRepo {
	return &itemRepository{Base{DB: db}}
}

const itemUpsert = ` insert into order_items(order_id, product_id, quantity, unit_price, discount, shipping_cost) values (?, ?, ?, ?, ?, ?) on duplicate key update quantity=values(quantity), unit_price=values(unit_price), discount=values(discount), shipping_cost=values(shipping_cost)`

func (r *itemRepository) Upsert(
	ctx context.Context,
	orderID, prodID string,
	qty int,
	price, disc, ship float64,
) error {
	return r.Exec(ctx, itemUpsert, orderID, prodID, qty, price, disc, ship)
}
