package store_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/teexue/nexa/core/store"
)

func openTestDB(t *testing.T) *store.DB {
	t.Helper()
	db, err := store.Open(t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func kanbanRow(id, userID, status string, priority int, createdAt time.Time) *store.KanbanRow {
	return &store.KanbanRow{
		ID:        id,
		UserID:    userID,
		Title:     "title-" + id,
		Prompt:    "do something",
		Agent:     "agt_demo",
		Status:    status,
		Priority:  priority,
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}
}

func TestKanban_CRUD(t *testing.T) {
	db := openTestDB(t)
	now := time.Now().UTC()

	row := kanbanRow("kb_1", "usr_local", "pending", 2, now)
	require.NoError(t, db.SaveKanban(row))

	got, err := db.GetKanban("kb_1")
	require.NoError(t, err)
	assert.Equal(t, "kb_1", got.ID)
	assert.Equal(t, "usr_local", got.UserID)
	assert.Equal(t, "pending", got.Status)

	// Upsert updates in place.
	got.Status = "review"
	got.Result = "done result"
	require.NoError(t, db.SaveKanban(got))
	reloaded, err := db.GetKanban("kb_1")
	require.NoError(t, err)
	assert.Equal(t, "review", reloaded.Status)
	assert.Equal(t, "done result", reloaded.Result)

	// List filters by user and sorts by created_at.
	older := kanbanRow("kb_2", "usr_local", "pending", 1, now.Add(-time.Hour))
	require.NoError(t, db.SaveKanban(older))
	require.NoError(t, db.SaveKanban(kanbanRow("kb_3", "usr_other", "pending", 1, now)))
	rows, err := db.ListKanban("usr_local")
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, "kb_2", rows[0].ID)
	assert.Equal(t, "kb_1", rows[1].ID)

	// Delete.
	require.NoError(t, db.DeleteKanban("kb_1"))
	_, err = db.GetKanban("kb_1")
	require.ErrorIs(t, err, store.ErrKanbanNotFound)
	require.ErrorIs(t, db.DeleteKanban("kb_1"), store.ErrKanbanNotFound)
}

func TestKanban_GetNotFound(t *testing.T) {
	db := openTestDB(t)
	_, err := db.GetKanban("kb_missing")
	require.ErrorIs(t, err, store.ErrKanbanNotFound)
}

func TestKanban_ClaimNextPending(t *testing.T) {
	db := openTestDB(t)
	now := time.Now().UTC()

	// Highest priority wins; ties break on oldest created_at.
	require.NoError(t, db.SaveKanban(kanbanRow("kb_low", "usr_local", "pending", 1, now.Add(-2*time.Hour))))
	require.NoError(t, db.SaveKanban(kanbanRow("kb_new", "usr_local", "pending", 3, now)))
	require.NoError(t, db.SaveKanban(kanbanRow("kb_old", "usr_local", "pending", 3, now.Add(-time.Hour))))
	require.NoError(t, db.SaveKanban(kanbanRow("kb_done", "usr_local", "done", 3, now.Add(-3*time.Hour))))
	require.NoError(t, db.SaveKanban(kanbanRow("kb_other", "usr_other", "pending", 3, now.Add(-4*time.Hour))))

	for _, wantID := range []string{"kb_old", "kb_new", "kb_low"} {
		claimed, err := db.ClaimNextPending("usr_local")
		require.NoError(t, err)
		assert.Equal(t, wantID, claimed.ID)
		assert.Equal(t, "running", claimed.Status)
	}

	// Empty for this user now.
	_, err := db.ClaimNextPending("usr_local")
	require.ErrorIs(t, err, store.ErrNoPending)

	// The other user's item is still claimable.
	claimed, err := db.ClaimNextPendingAny()
	require.NoError(t, err)
	assert.Equal(t, "kb_other", claimed.ID)

	// Claim persists the running status.
	reloaded, err := db.GetKanban(claimed.ID)
	require.NoError(t, err)
	assert.Equal(t, "running", reloaded.Status)

	_, err = db.ClaimNextPendingAny()
	require.ErrorIs(t, err, store.ErrNoPending)
}
