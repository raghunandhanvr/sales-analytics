package repository

import (
	"context"
)

type Base struct{ DB Database }

func (b Base) Exec(
	ctx context.Context,
	q string,
	args ...any,
) error {
	_, err := b.DB.ExecContext(ctx, q, args...)
	return err
}
