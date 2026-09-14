package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/teexue/nexakit/event"
	"github.com/teexue/common-agent/core/kanban"
	"github.com/teexue/nexakit/loop"
	"github.com/teexue/nexakit/session"
	"github.com/teexue/common-agent/core/store"
)

// kanbanResultMaxLen caps the aggregated run result stored on an item.
const kanbanResultMaxLen = 4000

// CreateKanbanRequest is the DTO for creating a kanban item.
type CreateKanbanRequest struct {
	Title    string     `json:"title"`
	Prompt   string     `json:"prompt"`
	Agent    string     `json:"agent"`
	WorkDir  string     `json:"workdir,omitempty"`
	Priority int        `json:"priority,omitempty"`
	Tags     []string   `json:"tags,omitempty"`
	DueAt    *time.Time `json:"due_at,omitempty"`
}

// UpdateKanbanRequest is the DTO for patching a kanban item. Nil fields are
// left unchanged.
type UpdateKanbanRequest struct {
	Title    *string    `json:"title,omitempty"`
	Prompt   *string    `json:"prompt,omitempty"`
	Agent    *string    `json:"agent,omitempty"`
	WorkDir  *string    `json:"workdir,omitempty"`
	Priority *int       `json:"priority,omitempty"`
	Tags     *[]string  `json:"tags,omitempty"`
	DueAt    *time.Time `json:"due_at,omitempty"`
}

// ListKanbanItems returns kanban items owned by userID.
func (s *Service) ListKanbanItems(userID string) ([]store.KanbanRow, error) {
	if s.StateDB == nil {
		return nil, fmt.Errorf("kanban persistence not configured")
	}
	return s.StateDB.ListKanban(userID)
}

// GetKanbanItem returns a single kanban item after verifying ownership.
func (s *Service) GetKanbanItem(id, userID string) (*store.KanbanRow, error) {
	return s.loadKanbanForUser(id, userID)
}

// CreateKanbanItem validates and persists a new pending kanban item.
func (s *Service) CreateKanbanItem(userID string, req CreateKanbanRequest) (*store.KanbanRow, error) {
	if s.StateDB == nil {
		return nil, fmt.Errorf("kanban persistence not configured")
	}
	if strings.TrimSpace(req.Title) == "" {
		return nil, &ArgError{Field: "title", Message: "title is required"}
	}
	if strings.TrimSpace(req.Prompt) == "" {
		return nil, &ArgError{Field: "prompt", Message: "prompt is required"}
	}
	if strings.TrimSpace(req.Agent) == "" {
		return nil, &ArgError{Field: "agent", Message: "agent is required"}
	}
	priority, err := normalizeKanbanPriority(req.Priority)
	if err != nil {
		return nil, err
	}
	tagsJSON, err := marshalKanbanTags(req.Tags)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	row := &store.KanbanRow{
		ID:        kanban.NewID(),
		UserID:    userID,
		Title:     req.Title,
		Prompt:    req.Prompt,
		Agent:     req.Agent,
		WorkDir:   req.WorkDir,
		Status:    kanban.StatusPending,
		Priority:  priority,
		TagsJSON:  tagsJSON,
		DueAt:     req.DueAt,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.StateDB.SaveKanban(row); err != nil {
		return nil, fmt.Errorf("save kanban item: %w", err)
	}
	return row, nil
}

// UpdateKanbanItem patches an item. Only pending or review items are editable.
func (s *Service) UpdateKanbanItem(id, userID string, req UpdateKanbanRequest) (*store.KanbanRow, error) {
	row, err := s.loadKanbanForUser(id, userID)
	if err != nil {
		return nil, err
	}
	if row.Status != kanban.StatusPending && row.Status != kanban.StatusReview {
		return nil, &ArgError{Field: "status", Message: fmt.Sprintf("cannot edit item in status %q", row.Status)}
	}
	if req.Title != nil {
		if strings.TrimSpace(*req.Title) == "" {
			return nil, &ArgError{Field: "title", Message: "title is required"}
		}
		row.Title = *req.Title
	}
	if req.Prompt != nil {
		if strings.TrimSpace(*req.Prompt) == "" {
			return nil, &ArgError{Field: "prompt", Message: "prompt is required"}
		}
		row.Prompt = *req.Prompt
	}
	if req.Agent != nil {
		if strings.TrimSpace(*req.Agent) == "" {
			return nil, &ArgError{Field: "agent", Message: "agent is required"}
		}
		row.Agent = *req.Agent
	}
	if req.WorkDir != nil {
		row.WorkDir = *req.WorkDir
	}
	if req.Priority != nil {
		priority, err := normalizeKanbanPriority(*req.Priority)
		if err != nil {
			return nil, err
		}
		row.Priority = priority
	}
	if req.Tags != nil {
		tagsJSON, err := marshalKanbanTags(*req.Tags)
		if err != nil {
			return nil, err
		}
		row.TagsJSON = tagsJSON
	}
	if req.DueAt != nil {
		row.DueAt = req.DueAt
	}
	return row, s.saveKanban(row)
}

// DeleteKanbanItem removes an item after verifying ownership.
func (s *Service) DeleteKanbanItem(id, userID string) error {
	if _, err := s.loadKanbanForUser(id, userID); err != nil {
		return err
	}
	return s.StateDB.DeleteKanban(id)
}

// ApproveKanbanItem marks a review item as done.
func (s *Service) ApproveKanbanItem(id, userID string) (*store.KanbanRow, error) {
	row, err := s.loadKanbanForUser(id, userID)
	if err != nil {
		return nil, err
	}
	if row.Status != kanban.StatusReview {
		return nil, &ArgError{Field: "status", Message: fmt.Sprintf("cannot approve item in status %q", row.Status)}
	}
	row.Status = kanban.StatusDone
	return row, s.saveKanban(row)
}

// RejectKanbanItem sends a review item back to pending with feedback.
func (s *Service) RejectKanbanItem(id, userID, feedback string) (*store.KanbanRow, error) {
	row, err := s.loadKanbanForUser(id, userID)
	if err != nil {
		return nil, err
	}
	if row.Status != kanban.StatusReview {
		return nil, &ArgError{Field: "status", Message: fmt.Sprintf("cannot reject item in status %q", row.Status)}
	}
	row.Status = kanban.StatusPending
	row.Feedback = feedback
	return row, s.saveKanban(row)
}

// RequeueKanbanItem resets a failed item to pending with zero attempts.
func (s *Service) RequeueKanbanItem(id, userID string) (*store.KanbanRow, error) {
	row, err := s.loadKanbanForUser(id, userID)
	if err != nil {
		return nil, err
	}
	if row.Status != kanban.StatusFailed {
		return nil, &ArgError{Field: "status", Message: fmt.Sprintf("cannot requeue item in status %q", row.Status)}
	}
	row.Status = kanban.StatusPending
	row.Attempts = 0
	row.Feedback = ""
	return row, s.saveKanban(row)
}

// KanbanRunner returns a kanban.Runner that executes items via
// PrepareRun + loop.Run, aggregating text output as the item result.
func (s *Service) KanbanRunner() kanban.Runner {
	return func(ctx context.Context, item *store.KanbanRow) (string, string, error) {
		prompt := item.Prompt
		if item.Feedback != "" {
			prompt += "\n\n[上次审核反馈] " + item.Feedback
		}
		result, err := s.PrepareRun(ctx, RunRequest{
			Agent:     item.Agent,
			Prompt:    prompt,
			WorkDir:   item.WorkDir,
			SessionID: item.SessionID, // retries continue the same session
			Source:    "kanban",
		}, nil)
		if err != nil && item.SessionID != "" && errors.Is(err, session.ErrNotFound) {
			// The stored session is gone (e.g. deleted manually) — start a
			// fresh one instead of failing the task.
			result, err = s.PrepareRun(ctx, RunRequest{
				Agent:   item.Agent,
				Prompt:  prompt,
				WorkDir: item.WorkDir,
				Source:  "kanban",
			}, nil)
		}
		if err != nil {
			return "", "", err
		}
		defer func() {
			result.Cleanup(s.Registry)
		}()

		// Kanban tasks run unattended: lift the agent's turn cap so long
		// tasks are not killed mid-work. (MaxTurns 0 = run until the model
		// stops calling tools.)
		result.Config.Agent.MaxTurns = 0

		// Persist the session id up front so the UI can follow the run live.
		item.SessionID = result.Session.ID
		if s.StateDB != nil {
			item.UpdatedAt = time.Now().UTC()
			_ = s.StateDB.SaveKanban(item)
		}
		// Save the session right away (the loop only persists it at run end).
		// Attribute it to the kanban item's owner rather than the default
		// local user, and mark it so it stays out of the conversation
		// session list.
		if item.UserID != "" {
			result.Session.UserID = item.UserID
		}
		result.Session.SetMetadata(session.MetadataKeySource, session.SourceKanban)
		if s.Store != nil {
			_ = s.Store.Save(result.Session)
		}

		events, err := loop.Run(ctx, result.Config)
		if err != nil {
			return "", result.Session.ID, err
		}
		var sb strings.Builder
		var runErr error
		for ev := range events {
			switch ev.Type {
			case event.TypeTextDelta:
				sb.WriteString(ev.Content)
			case event.TypeError:
				msg := ev.Message
				if msg == "" {
					msg = ev.Content
				}
				if msg != "" {
					runErr = fmt.Errorf("%s", msg)
				}
			}
		}
		text := sb.String()
		if runes := []rune(text); len(runes) > kanbanResultMaxLen {
			text = string(runes[:kanbanResultMaxLen])
		}
		return text, result.Session.ID, runErr
	}
}

// loadKanbanForUser loads an item and verifies ownership.
func (s *Service) loadKanbanForUser(id, userID string) (*store.KanbanRow, error) {
	if s.StateDB == nil {
		return nil, fmt.Errorf("kanban persistence not configured")
	}
	if id == "" {
		return nil, &ArgError{Field: "id", Message: "kanban item id is required"}
	}
	row, err := s.StateDB.GetKanban(id)
	if err != nil {
		return nil, err
	}
	if userID != "" && row.UserID != userID {
		return nil, fmt.Errorf("%w: %s", store.ErrKanbanNotFound, id)
	}
	return row, nil
}

func (s *Service) saveKanban(row *store.KanbanRow) error {
	row.UpdatedAt = time.Now().UTC()
	if err := s.StateDB.SaveKanban(row); err != nil {
		return fmt.Errorf("save kanban item: %w", err)
	}
	return nil
}

func normalizeKanbanPriority(p int) (int, error) {
	if p == 0 {
		return kanban.PriorityMedium, nil
	}
	if p < kanban.PriorityLow || p > kanban.PriorityHigh {
		return 0, &ArgError{Field: "priority", Message: "priority must be 1 (low), 2 (medium) or 3 (high)"}
	}
	return p, nil
}

func marshalKanbanTags(tags []string) (string, error) {
	if len(tags) == 0 {
		return "", nil
	}
	b, err := json.Marshal(tags)
	if err != nil {
		return "", fmt.Errorf("marshal tags: %w", err)
	}
	return string(b), nil
}
