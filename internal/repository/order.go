package repository

import (
	"context"
	"time"
)

type orderRepository struct {
	Base
}

func NewOrderRepo(db Database) OrderRepo {
	return &orderRepository{Base{DB: db}}
}

const orderUpsert = `insert into orders(id, customer_id, order_date, total_amount) values (?, ?, ?, ?) on duplicate key update total_amount=values(total_amount)`

func (r *orderRepository) Upsert(
	ctx context.Context,
	id, custID string,
	date time.Time,
	total float64,
) error {
	return r.Exec(ctx, orderUpsert, id, custID, date, total)
}
