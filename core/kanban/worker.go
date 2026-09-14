package kanban

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/teexue/nexa/core/store"
)

// Runner executes one kanban item to completion and returns the aggregated
// result text plus the session id used for the run.
type Runner func(ctx context.Context, item *store.KanbanRow) (result string, sessionID string, err error)

// Worker polls the store for pending kanban items and executes them via Runner.
type Worker struct {
	store    *store.DB
	runner   Runner
	logger   *slog.Logger
	interval time.Duration

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// WorkerConfig configures a Worker.
type WorkerConfig struct {
	Store     *store.DB
	Runner    Runner
	Logger    *slog.Logger
	TickEvery time.Duration
}

// NewWorker creates a Worker.
func NewWorker(cfg WorkerConfig) *Worker {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	tick := cfg.TickEvery
	if tick <= 0 {
		tick = 5 * time.Second
	}
	return &Worker{
		store:    cfg.Store,
		runner:   cfg.Runner,
		logger:   logger,
		interval: tick,
	}
}

// Start begins the polling loop until ctx is cancelled.
func (w *Worker) Start(ctx context.Context) {
	ctx, w.cancel = context.WithCancel(ctx)
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()
		w.tick(ctx)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				w.tick(ctx)
			}
		}
	}()
}

// Stop cancels the worker and waits for the current run to finish.
func (w *Worker) Stop() {
	if w.cancel != nil {
		w.cancel()
	}
	w.wg.Wait()
}

func (w *Worker) tick(ctx context.Context) {
	if err := w.processOne(ctx); err != nil {
		w.logger.Warn("log.kanban.process_failed", "error", err)
	}
}

// processOne claims one pending item and runs it. Returns nil when there is
// nothing to do.
func (w *Worker) processOne(ctx context.Context) error {
	row, err := w.store.ClaimNextPendingAny()
	if errors.Is(err, store.ErrNoPending) {
		return nil
	}
	if err != nil {
		return err
	}

	result, sessionID, runErr := w.runner(ctx, row)

	now := time.Now().UTC()
	row.UpdatedAt = now
	if runErr == nil {
		row.Status = StatusReview
		row.Result = result
		row.SessionID = sessionID
		row.Feedback = ""
		row.FinishedAt = &now
	} else {
		row.Attempts++
		row.Feedback = runErr.Error()
		if row.Attempts >= MaxAttempts {
			row.Status = StatusFailed
			row.FinishedAt = &now
		} else {
			row.Status = StatusPending
		}
	}
	if err := w.store.SaveKanban(row); err != nil {
		return err
	}
	w.logger.Info("log.kanban.item_processed", "kanban_id", row.ID, "status", row.Status, "attempts", row.Attempts)
	return nil
}
