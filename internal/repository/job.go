package repository

import (
	"context"

	"sales-analytics/internal/models"
	"sales-analytics/pkg/orm"
)

type jobRepo struct{ store *orm.Store }

func NewJobRepo(store *orm.Store) JobRepository {
	return &jobRepo{store: store}
}

func (r *jobRepo) Insert(
	ctx context.Context,
	id string,
) {
	r.store.DB.ExecContext(ctx, "insert into ingestion_jobs(job_id,status) values(?,?)", id, "running")
}

func (r *jobRepo) SetFailed(
	ctx context.Context,
	id, msg string,
) {
	r.store.DB.ExecContext(ctx, "update ingestion_jobs set status='failed',error_message=? where job_id=?", msg, id)
}

func (r *jobRepo) SetCompleted(
	ctx context.Context,
	id string,
	rows int,
) {
	r.store.DB.ExecContext(ctx, "update ingestion_jobs set status='completed',total_rows=?,processed_rows=? where job_id=?",
		rows, rows, id)
}

func (r *jobRepo) Bump(
	ctx context.Context,
	id string,
	rows int,
) {
	r.store.DB.ExecContext(ctx, "update ingestion_jobs set processed_rows=? where job_id=?", rows, id)
}

func (r *jobRepo) Get(
	ctx context.Context,
	id string,
) (models.IngestionJob, error) {
	var m models.IngestionJob
	err := r.store.DB.QueryRowContext(ctx, `select job_id,status,total_rows,processed_rows,
		coalesce(error_message,''),created_at,updated_at from ingestion_jobs where job_id=?`, id).
		Scan(&m.JobID, &m.Status, &m.TotalRows, &m.ProcessedRows, &m.ErrorMessage, &m.CreatedAt, &m.UpdatedAt)
	return m, err
}
