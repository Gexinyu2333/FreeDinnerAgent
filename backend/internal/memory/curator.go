package memory

import (
	"context"

	"freedinner/backend/internal/store"
)

func (m *Manager) EnqueueCuratorJob(ctx context.Context, userID, jobType string, payload []byte) (store.CuratorJob, error) {
	if m.profiles == nil {
		return store.CuratorJob{}, nil
	}
	return m.profiles.CreateCuratorJob(ctx, store.CuratorJobCreate{
		UserID:  userID,
		JobType: jobType,
		Payload: payload,
	})
}
