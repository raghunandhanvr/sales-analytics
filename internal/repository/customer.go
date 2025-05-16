package repository

import (
	"context"
	"strings"

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

func (r *customerRepository) BulkUpsert(
	ctx context.Context,
	customers []models.Customer,
) (int, error) {
	if len(customers) == 0 {
		return 0, nil
	}

	valueStrings := make([]string, 0, len(customers))
	valueArgs := make([]interface{}, 0, len(customers)*5)

	for _, c := range customers {
		valueStrings = append(valueStrings, "(?, ?, ?, ?, ?)")
		valueArgs = append(valueArgs, c.ID, c.Name, c.Email, c.Region, c.Address)
	}

	stmt := `insert into customers(id, name, email, region, address) values ` +
		strings.Join(valueStrings, ",") +
		` on duplicate key update 
		name=values(name),
		region=values(region),
		address=values(address)`

	result, err := r.DB.ExecContext(ctx, stmt, valueArgs...)
	if err != nil {
		return 0, err
	}

	affected, _ := result.RowsAffected()
	return int(affected), nil
}
