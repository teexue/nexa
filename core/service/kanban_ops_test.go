package service_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/teexue/nexa/core/kanban"
	"github.com/teexue/nexa/core/service"
	"github.com/teexue/nexa/core/store"
)

func newKanbanService(t *testing.T) (*service.Service, *store.DB) {
	t.Helper()
	db, err := store.Open(t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return service.New(service.ServiceConfig{StateDB: db}), db
}

func createItem(t *testing.T, svc *service.Service, userID string) *store.KanbanRow {
	t.Helper()
	row, err := svc.CreateKanbanItem(userID, service.CreateKanbanRequest{
		Title:  "task",
		Prompt: "do it",
		Agent:  "agt_demo",
	})
	require.NoError(t, err)
	return row
}

func setStatus(t *testing.T, db *store.DB, id, status string) {
	t.Helper()
	row, err := db.GetKanban(id)
	require.NoError(t, err)
	row.Status = status
	require.NoError(t, db.SaveKanban(row))
}

func TestCreateKanbanItem_Validation(t *testing.T) {
	svc, _ := newKanbanService(t)

	tests := []struct {
		name string
		req  service.CreateKanbanRequest
	}{
		{name: "missing title", req: service.CreateKanbanRequest{Prompt: "p", Agent: "a"}},
		{name: "missing prompt", req: service.CreateKanbanRequest{Title: "t", Agent: "a"}},
		{name: "missing agent", req: service.CreateKanbanRequest{Title: "t", Prompt: "p"}},
		{name: "bad priority", req: service.CreateKanbanRequest{Title: "t", Prompt: "p", Agent: "a", Priority: 9}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.CreateKanbanItem("usr_local", tt.req)
			require.Error(t, err)
			var argErr *service.ArgError
			require.ErrorAs(t, err, &argErr)
		})
	}
}

func TestCreateKanbanItem_Defaults(t *testing.T) {
	svc, _ := newKanbanService(t)
	due := time.Now().UTC().Add(24 * time.Hour)

	row, err := svc.CreateKanbanItem("usr_local", service.CreateKanbanRequest{
		Title:  "task",
		Prompt: "do it",
		Agent:  "agt_demo",
		Tags:   []string{"a", "b"},
		DueAt:  &due,
	})
	require.NoError(t, err)
	assert.Equal(t, kanban.StatusPending, row.Status)
	assert.Equal(t, kanban.PriorityMedium, row.Priority)
	assert.Equal(t, "usr_local", row.UserID)
	assert.Equal(t, `["a","b"]`, row.TagsJSON)
	require.NotNil(t, row.DueAt)
}

func TestKanbanItem_ApproveRejectFlow(t *testing.T) {
	svc, db := newKanbanService(t)
	item := createItem(t, svc, "usr_local")

	// Approve only works from review.
	_, err := svc.ApproveKanbanItem(item.ID, "usr_local")
	require.Error(t, err)

	setStatus(t, db, item.ID, kanban.StatusReview)
	_, err = svc.RejectKanbanItem(item.ID, "usr_local", "needs more detail")
	require.NoError(t, err)
	row, err := db.GetKanban(item.ID)
	require.NoError(t, err)
	assert.Equal(t, kanban.StatusPending, row.Status)
	assert.Equal(t, "needs more detail", row.Feedback)

	// Reject only works from review.
	_, err = svc.RejectKanbanItem(item.ID, "usr_local", "again")
	require.Error(t, err)

	setStatus(t, db, item.ID, kanban.StatusReview)
	approved, err := svc.ApproveKanbanItem(item.ID, "usr_local")
	require.NoError(t, err)
	assert.Equal(t, kanban.StatusDone, approved.Status)
}

func TestKanbanItem_Requeue(t *testing.T) {
	svc, db := newKanbanService(t)
	item := createItem(t, svc, "usr_local")

	// Requeue only works from failed.
	_, err := svc.RequeueKanbanItem(item.ID, "usr_local")
	require.Error(t, err)

	setStatus(t, db, item.ID, kanban.StatusFailed)
	row, err := db.GetKanban(item.ID)
	require.NoError(t, err)
	row.Attempts = kanban.MaxAttempts
	row.Feedback = "boom"
	require.NoError(t, db.SaveKanban(row))

	requeued, err := svc.RequeueKanbanItem(item.ID, "usr_local")
	require.NoError(t, err)
	assert.Equal(t, kanban.StatusPending, requeued.Status)
	assert.Equal(t, 0, requeued.Attempts)
	assert.Empty(t, requeued.Feedback)
}

func TestKanbanItem_UpdateRestrictions(t *testing.T) {
	svc, db := newKanbanService(t)
	item := createItem(t, svc, "usr_local")

	newTitle := "renamed"
	updated, err := svc.UpdateKanbanItem(item.ID, "usr_local", service.UpdateKanbanRequest{Title: &newTitle})
	require.NoError(t, err)
	assert.Equal(t, "renamed", updated.Title)

	// Running items are not editable.
	setStatus(t, db, item.ID, kanban.StatusRunning)
	_, err = svc.UpdateKanbanItem(item.ID, "usr_local", service.UpdateKanbanRequest{Title: &newTitle})
	require.Error(t, err)
}

func TestKanbanItem_Ownership(t *testing.T) {
	svc, _ := newKanbanService(t)
	item := createItem(t, svc, "usr_local")

	_, err := svc.ApproveKanbanItem(item.ID, "usr_other")
	require.ErrorIs(t, err, store.ErrKanbanNotFound)

	require.ErrorIs(t, svc.DeleteKanbanItem(item.ID, "usr_other"), store.ErrKanbanNotFound)
	require.NoError(t, svc.DeleteKanbanItem(item.ID, "usr_local"))
}
