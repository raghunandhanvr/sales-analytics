package repository

import (
	"context"
	"sales-analytics/internal/models"
	"strings"
	"time"
)

type orderRepository struct {
	Base
}

func NewOrderRepo(db Database) OrderRepo {
	return &orderRepository{Base{DB: db}}
}

const orderUpsert = `insert into orders(id, customer_id, order_date, total_amount) values (?, ?, ?, ?) on duplicate key update customer_id=values(customer_id), order_date=values(order_date), total_amount=values(total_amount)`

func (r *orderRepository) Upsert(
	ctx context.Context,
	id, custID string,
	date time.Time,
	total float64,
) error {
	return r.Exec(ctx, orderUpsert, id, custID, date, total)
}

func (r *orderRepository) BulkUpsert(
	ctx context.Context,
	orderParams []models.Order,
) (int, error) {
	if len(orderParams) == 0 {
		return 0, nil
	}

	valueStrings := make([]string, 0, len(orderParams))
	valueArgs := make([]interface{}, 0, len(orderParams)*4)

	for _, o := range orderParams {
		valueStrings = append(valueStrings, "(?, ?, ?, ?)")
		valueArgs = append(valueArgs, o.ID, o.CustomerID, o.OrderDate, o.TotalAmount)
	}

	stmt := `insert into orders(id, customer_id, order_date, total_amount) values ` +
		strings.Join(valueStrings, ",") +
		` on duplicate key update 
		customer_id=values(customer_id),
		order_date=values(order_date),
		total_amount=values(total_amount)`

	result, err := r.DB.ExecContext(ctx, stmt, valueArgs...)
	if err != nil {
		return 0, err
	}

	affected, _ := result.RowsAffected()
	return int(affected), nil
}
