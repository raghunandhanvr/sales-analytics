package models

import "time"

type IngestionJob struct {
	JobID         string    `json:"job_id"`
	Status        string    `json:"status"`
	TotalRows     int64     `json:"total_rows"`
	ProcessedRows int64     `json:"processed_rows"`
	ErrorMessage  string    `json:"error_message,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
