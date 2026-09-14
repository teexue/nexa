package kanban_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/teexue/nexa/core/kanban"
	"github.com/teexue/nexa/core/store"
)

func openWorkerDB(t *testing.T) *store.DB {
	t.Helper()
	db, err := store.Open(t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func savePending(t *testing.T, db *store.DB, id string) {
	t.Helper()
	now := time.Now().UTC()
	require.NoError(t, db.SaveKanban(&store.KanbanRow{
		ID:        id,
		UserID:    "usr_owner",
		Title:     "item " + id,
		Prompt:    "run it",
		Agent:     "agt_demo",
		Status:    kanban.StatusPending,
		Priority:  kanban.PriorityMedium,
		CreatedAt: now,
		UpdatedAt: now,
	}))
}

func startWorker(db *store.DB, runner kanban.Runner) *kanban.Worker {
	w := kanban.NewWorker(kanban.WorkerConfig{
		Store:     db,
		Runner:    runner,
		TickEvery: 5 * time.Millisecond,
	})
	w.Start(context.Background())
	return w
}

func waitStatus(t *testing.T, db *store.DB, id, status string) *store.KanbanRow {
	t.Helper()
	var row *store.KanbanRow
	require.Eventually(t, func() bool {
		var err error
		row, err = db.GetKanban(id)
		return err == nil && row.Status == status
	}, 5*time.Second, 10*time.Millisecond)
	return row
}

func TestWorker_SuccessMovesToReview(t *testing.T) {
	db := openWorkerDB(t)
	savePending(t, db, "kb_ok")

	w := startWorker(db, func(_ context.Context, item *store.KanbanRow) (string, string, error) {
		return "result text", "sess_1", nil
	})
	defer w.Stop()

	row := waitStatus(t, db, "kb_ok", kanban.StatusReview)
	assert.Equal(t, "result text", row.Result)
	assert.Equal(t, "sess_1", row.SessionID)
	assert.Equal(t, 0, row.Attempts)
	require.NotNil(t, row.FinishedAt)
}

func TestWorker_FailureReturnsToPending(t *testing.T) {
	db := openWorkerDB(t)
	savePending(t, db, "kb_fail")

	// First attempt fails; the item must return to pending and be retried.
	var callCount atomic.Int32
	w := startWorker(db, func(_ context.Context, item *store.KanbanRow) (string, string, error) {
		if callCount.Add(1) == 1 {
			return "", "", errors.New("boom")
		}
		return "recovered", "sess_2", nil
	})
	defer w.Stop()

	// Reaching review proves the item went back to pending after the error.
	row := waitStatus(t, db, "kb_fail", kanban.StatusReview)
	assert.Equal(t, int32(2), callCount.Load())
	assert.Equal(t, 1, row.Attempts)
	assert.Equal(t, "recovered", row.Result)
	assert.Empty(t, row.Feedback)
}

func TestWorker_MaxAttemptsMarksFailed(t *testing.T) {
	db := openWorkerDB(t)
	savePending(t, db, "kb_doom")

	w := startWorker(db, func(_ context.Context, item *store.KanbanRow) (string, string, error) {
		return "", "", errors.New("always fails")
	})
	defer w.Stop()

	row := waitStatus(t, db, "kb_doom", kanban.StatusFailed)
	assert.Equal(t, kanban.MaxAttempts, row.Attempts)
	assert.Equal(t, "always fails", row.Feedback)
	require.NotNil(t, row.FinishedAt)
}

func TestWorker_IdleWhenNoPending(t *testing.T) {
	db := openWorkerDB(t)
	ran := make(chan struct{}, 1)
	w := startWorker(db, func(_ context.Context, item *store.KanbanRow) (string, string, error) {
		ran <- struct{}{}
		return "", "", nil
	})
	defer w.Stop()

	select {
	case <-ran:
		t.Fatal("runner should not run without pending items")
	case <-time.After(100 * time.Millisecond):
	}
}
