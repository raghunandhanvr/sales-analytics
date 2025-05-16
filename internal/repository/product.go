package repository

import (
	"context"
	"strings"

	"sales-analytics/internal/models"
)

type productRepository struct {
	Base
}

func NewProductRepo(db Database) ProductRepo {
	return &productRepository{Base{DB: db}}
}

const prodUpsert = `insert into products(id, name, category, unit_price) values (?, ?, ?, ?) on duplicate key update name=values(name), category=values(category), unit_price=values(unit_price)`

func (r *productRepository) Upsert(
	ctx context.Context,
	p models.Product,
) error {
	return r.Exec(ctx, prodUpsert, p.ID, p.Name, p.Category, p.UnitPrice)
}

func (r *productRepository) BulkUpsert(
	ctx context.Context,
	products []models.Product,
) (int, error) {
	if len(products) == 0 {
		return 0, nil
	}

	valueStrings := make([]string, 0, len(products))
	valueArgs := make([]interface{}, 0, len(products)*4)

	for _, p := range products {
		valueStrings = append(valueStrings, "(?, ?, ?, ?)")
		valueArgs = append(valueArgs, p.ID, p.Name, p.Category, p.UnitPrice)
	}

	stmt := `insert into products(id, name, category, unit_price) values ` +
		strings.Join(valueStrings, ",") +
		` on duplicate key update 
		name=values(name),
		category=values(category),
		unit_price=values(unit_price)`

	result, err := r.DB.ExecContext(ctx, stmt, valueArgs...)
	if err != nil {
		return 0, err
	}

	affected, _ := result.RowsAffected()
	return int(affected), nil
}
