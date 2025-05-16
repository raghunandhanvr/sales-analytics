package repository

import (
	"context"

	"sales-analytics/internal/models"
)

type customerRepository struct {
	Base
}

func NewCustomerRepo(db Database) CustomerRepo {
	return &customerRepository{Base{DB: db}}
}

const custUpsert = ` insert into customers(id, name, email, region, address) values (?, ?, ?, ?, ?) on duplicate key update name=values(name), region=values(region), address=values(address)`

func (r *customerRepository) Upsert(
	ctx context.Context,
	c models.Customer,
) error {
	return r.Exec(ctx, custUpsert, c.ID, c.Name, c.Email, c.Region, c.Address)
}
