package repository

import (
	"context"
	"sales-analytics/internal/models"
	"strings"
)

type itemRepository struct {
	Base
}

func NewItemRepo(db Database) ItemRepo {
	return &itemRepository{Base{DB: db}}
}

const itemUpsert = `insert into order_items(order_id, product_id, quantity, unit_price, discount, shipping_cost) values (?, ?, ?, ?, ?, ?) on duplicate key update quantity=values(quantity), unit_price=values(unit_price), discount=values(discount), shipping_cost=values(shipping_cost)`

func (r *itemRepository) Upsert(
	ctx context.Context,
	orderID, prodID string,
	qty int,
	price, disc, ship float64,
) error {
	return r.Exec(ctx, itemUpsert, orderID, prodID, qty, price, disc, ship)
}

func (r *itemRepository) BulkUpsert(
	ctx context.Context,
	itemParams []models.OrderItem,
) (int, error) {
	if len(itemParams) == 0 {
		return 0, nil
	}

	valueStrings := make([]string, 0, len(itemParams))
	valueArgs := make([]interface{}, 0, len(itemParams)*6)

	for _, item := range itemParams {
		valueStrings = append(valueStrings, "(?, ?, ?, ?, ?, ?)")
		valueArgs = append(valueArgs,
			item.OrderID,
			item.ProductID,
			item.Quantity,
			item.UnitPrice,
			item.Discount,
			item.ShippingCost)
	}

	stmt := `insert into order_items(order_id, product_id, quantity, unit_price, discount, shipping_cost) values ` +
		strings.Join(valueStrings, ",") +
		` on duplicate key update 
		quantity=values(quantity),
		unit_price=values(unit_price),
		discount=values(discount),
		shipping_cost=values(shipping_cost)`

	result, err := r.DB.ExecContext(ctx, stmt, valueArgs...)
	if err != nil {
		return 0, err
	}

	affected, _ := result.RowsAffected()
	return int(affected), nil
}
