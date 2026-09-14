package service

import (
	"fmt"

	"github.com/teexue/nexakit/session"
	"github.com/teexue/common-agent/core/store"
)

// ListSessions returns metadata for sessions owned by userID. Sessions
// created by kanban or sub-agent runs are excluded from the conversation list.
func (s *Service) ListSessions(userID string) ([]session.SessionMeta, error) {
	if s.Store == nil {
		return nil, fmt.Errorf("session persistence not configured")
	}
	var metas []session.SessionMeta
	if gs, ok := s.Store.(*store.SessionStore); ok {
		var err error
		metas, err = gs.ListByUser(userID)
		if err != nil {
			return nil, err
		}
	} else {
		var err error
		metas, err = s.Store.List()
		if err != nil {
			return nil, err
		}
	}
	out := make([]session.SessionMeta, 0, len(metas))
	for _, m := range metas {
		if hiddenSessionSource(m.Metadata[session.MetadataKeySource]) {
			continue
		}
		uid := m.UserID
		if userID == "" || uid == userID {
			if s.Hub != nil && s.Hub.Running(m.ID) {
				m.Running = true
			}
			out = append(out, m)
		}
	}
	return out, nil
}

// LoadSession retrieves a session by ID for the given user.
func (s *Service) LoadSession(id, userID string) (*session.Session, error) {
	if s.Store == nil {
		return nil, fmt.Errorf("session persistence not configured")
	}
	if id == "" {
		return nil, fmt.Errorf("session id is required")
	}
	if gs, ok := s.Store.(*store.SessionStore); ok {
		return gs.LoadForUser(id, userID)
	}
	sess, err := s.Store.Load(id)
	if err != nil {
		return nil, err
	}
	if userID != "" && sess.UserID != userID {
		return nil, session.ErrNotFound
	}
	return sess, nil
}

// DeleteSession removes a session by ID for the given user.
func (s *Service) DeleteSession(id, userID string) error {
	if s.Store == nil {
		return fmt.Errorf("session persistence not configured")
	}
	if id == "" {
		return fmt.Errorf("session id is required")
	}
	if s.Hub != nil {
		s.Hub.AbortAndWait(id)
	}
	if gs, ok := s.Store.(*store.SessionStore); ok {
		return gs.DeleteForUser(id, userID)
	}
	if _, err := s.LoadSession(id, userID); err != nil {
		return err
	}
	return s.Store.Delete(id)
}

func hiddenSessionSource(src string) bool {
	return src == session.SourceKanban || src == session.SourceSubagent
}
