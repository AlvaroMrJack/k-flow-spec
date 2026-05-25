package runner

import (
	"context"
	"log/slog"
	"sync"

	"github.com/AlvaroMrJack/k-flow-spec/internal/config"
	"github.com/AlvaroMrJack/k-flow-spec/pkg/kapso"
	"github.com/AlvaroMrJack/k-flow-spec/pkg/spec"
)

type Scheduler struct {
	engine *Engine
	cfg    *config.KfsConfig
}

func NewScheduler(client *kapso.Client, cfg *config.KfsConfig) *Scheduler {
	return &Scheduler{
		engine: NewEngine(client, cfg),
		cfg:    cfg,
	}
}

func (s *Scheduler) RunAll(ctx context.Context, specs []*spec.Spec) []*Result {
	var mu sync.Mutex
	results := make([]*Result, 0, len(specs))

	maxConcurrent := s.cfg.RateLimit.MaxBurst
	if maxConcurrent <= 0 {
		maxConcurrent = 5
	}

	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup

	for _, sp := range specs {
		select {
		case <-ctx.Done():
			slog.Warn("context cancelled, stopping scheduler")
			return results
		case sem <- struct{}{}:
		}

		wg.Add(1)
		go func(spec *spec.Spec) {
			defer wg.Done()
			defer func() { <-sem }()

			result := s.engine.Run(ctx, spec)

			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(sp)
	}

	wg.Wait()
	return results
}
