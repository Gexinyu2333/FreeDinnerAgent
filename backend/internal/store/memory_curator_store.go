package store

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

func (s *MemoryStore) CreateCuratorJob(ctx context.Context, input CuratorJobCreate) (CuratorJob, error) {
	if len(input.Payload) == 0 {
		input.Payload = json.RawMessage(`{}`)
	}
	return scanCuratorJob(s.db.QueryRow(ctx, `
		INSERT INTO curator_jobs (id, user_id, job_type, payload)
		VALUES ($1, $2, $3, $4)
		RETURNING id, user_id, job_type, payload, status, error_message, created_at, started_at, finished_at
	`, uuid.NewString(), input.UserID, input.JobType, input.Payload))
}
