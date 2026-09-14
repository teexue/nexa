package store

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/teexue/nexakit/provider"
	"github.com/teexue/nexakit/session"
)

// SessionStore persists sessions in SQLite via GORM.
type SessionStore struct {
	db *DB
}

// NewSessionStore creates a GORM-backed session store.
func NewSessionStore(db *DB) *SessionStore {
	return &SessionStore{db: db}
}

// Save persists a session.
func (s *SessionStore) Save(sess *session.Session) error {
	msgs := sess.GetMessages()
	msgJSON, err := json.Marshal(msgs)
	if err != nil {
		return fmt.Errorf("marshal messages: %w", err)
	}
	meta := sess.GetMetadata()
	metaJSON := ""
	if len(meta) > 0 {
		b, err := json.Marshal(meta)
		if err != nil {
			return fmt.Errorf("marshal metadata: %w", err)
		}
		metaJSON = string(b)
	}
	userID := sess.UserID
	row := SessionRow{
		ID:           sess.ID,
		UserID:       userID,
		Agent:        sess.Agent,
		Title:        sess.GetTitle(),
		MessagesJSON: string(msgJSON),
		MetadataJSON: metaJSON,
		CreatedAt:    sess.CreatedAt,
		UpdatedAt:    sess.UpdatedAt,
	}
	if row.CreatedAt.IsZero() {
		row.CreatedAt = time.Now().UTC()
	}
	if row.UpdatedAt.IsZero() {
		row.UpdatedAt = time.Now().UTC()
	}
	return s.db.Save(&row).Error
}

// Load retrieves a session by ID.
func (s *SessionStore) Load(id string) (*session.Session, error) {
	var row SessionRow
	if err := s.db.Where("id = ?", id).First(&row).Error; err != nil {
		if isNotFound(err) {
			return nil, fmt.Errorf("%w: %s", session.ErrNotFound, id)
		}
		return nil, err
	}
	return rowToSession(row)
}

// List returns metadata for all sessions (caller filters by user if needed).
func (s *SessionStore) List() ([]session.SessionMeta, error) {
	return s.ListByUser("")
}

// ListByUser returns sessions for a user, or all when userID is empty.
func (s *SessionStore) ListByUser(userID string) ([]session.SessionMeta, error) {
	q := s.db.Model(&SessionRow{}).Order("updated_at desc")
	if userID != "" {
		q = q.Where("user_id = ?", userID)
	}
	var rows []SessionRow
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]session.SessionMeta, 0, len(rows))
	for _, r := range rows {
		meta := session.SessionMeta{
			ID: r.ID, UserID: r.UserID, Agent: r.Agent, Title: r.Title,
			CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
		}
		if r.MetadataJSON != "" {
			_ = json.Unmarshal([]byte(r.MetadataJSON), &meta.Metadata)
		}
		if meta.Title == "" {
			var msgs []provider.Message
			if json.Unmarshal([]byte(r.MessagesJSON), &msgs) == nil {
				meta.Title = titleFromProviderMessages(msgs)
			}
		}
		out = append(out, meta)
	}
	return out, nil
}

// Delete removes a session by ID.
func (s *SessionStore) Delete(id string) error {
	res := s.db.Where("id = ?", id).Delete(&SessionRow{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("%w: %s", session.ErrNotFound, id)
	}
	return nil
}

// LoadForUser loads a session and verifies ownership.
func (s *SessionStore) LoadForUser(id, userID string) (*session.Session, error) {
	sess, err := s.Load(id)
	if err != nil {
		return nil, err
	}
	if userID != "" && sess.UserID != "" && sess.UserID != userID {
		return nil, fmt.Errorf("%w: %s", session.ErrNotFound, id)
	}
	return sess, nil
}

// DeleteForUser deletes a session after verifying ownership.
func (s *SessionStore) DeleteForUser(id, userID string) error {
	if _, err := s.LoadForUser(id, userID); err != nil {
		return err
	}
	return s.Delete(id)
}

func rowToSession(row SessionRow) (*session.Session, error) {
	var msgs []provider.Message
	if row.MessagesJSON != "" {
		if err := json.Unmarshal([]byte(row.MessagesJSON), &msgs); err != nil {
			return nil, fmt.Errorf("parse messages: %w", err)
		}
	}
	meta := map[string]string{}
	if row.MetadataJSON != "" {
		if err := json.Unmarshal([]byte(row.MetadataJSON), &meta); err != nil {
			return nil, fmt.Errorf("parse metadata: %w", err)
		}
	}
	userID := row.UserID
	return &session.Session{
		ID:        row.ID,
		UserID:    userID,
		Agent:     row.Agent,
		Title:     row.Title,
		Messages:  msgs,
		Metadata:  meta,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}, nil
}

func titleFromProviderMessages(msgs []provider.Message) string {
	for _, m := range msgs {
		if m.Role == provider.RoleUser && !provider.IsToolImageUserMessage(m) {
			return session.TitleFromPrompt(m.Content)
		}
	}
	return ""
}

// Ensure SessionStore implements session.Store.
var _ session.Store = (*SessionStore)(nil)
