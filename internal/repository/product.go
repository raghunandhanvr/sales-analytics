package repository

import (
	"context"

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
