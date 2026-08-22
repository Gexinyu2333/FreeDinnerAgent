package scheduler

import (
	"context"
	"log"
	"time"
)

func (s *Service) StartWorker(ctx context.Context, interval time.Duration, limit int) {
	if interval <= 0 {
		interval = time.Minute
	}
	if limit <= 0 {
		limit = 20
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		s.runWorkerTick(ctx, limit)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.runWorkerTick(ctx, limit)
			}
		}
	}()
}

func (s *Service) runWorkerTick(ctx context.Context, limit int) {
	results, err := s.RunDue(ctx, time.Now(), limit)
	if err != nil {
		log.Printf("scheduled job worker: %v", err)
		return
	}
	for _, result := range results {
		if result.Error != nil {
			log.Printf("scheduled job worker: job %s %s: %s", result.JobID, result.Status, *result.Error)
		}
	}
}
