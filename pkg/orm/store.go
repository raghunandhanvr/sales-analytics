package orm

import "database/sql"

type Store struct{ DB *sql.DB }

func New(db *sql.DB) *Store { return &Store{DB: db} }

func (s *Store) Exec(
	query string,
	args ...any,
) error {
	_, err := s.DB.Exec(query, args...)
	return err
}

func (s *Store) Query(
	dest any,
	query string,
	args ...any,
) error {
	return s.DB.QueryRow(query, args...).Scan(dest)
}
