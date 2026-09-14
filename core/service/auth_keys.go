package service

import (
	"strings"
	"time"

	"github.com/teexue/nexa/core/store"
)

// CreateAPIKeyRequest is the input for generating a persisted API key.
type CreateAPIKeyRequest struct {
	UserID        string
	Name          string
	Scopes        []string
	ExpiresInDays int
}

// CreatedAPIKey holds the raw secret (returned once) and the stored record.
type CreatedAPIKey struct {
	Raw string
	Key store.APIKey
}

// ListAPIKeys returns redacted keys for a user. An empty list if auth is unset.
func (s *Service) ListAPIKeys(userID string) ([]store.APIKeyInfo, error) {
	db, err := s.state()
	if err != nil {
		return []store.APIKeyInfo{}, nil
	}
	return db.ListAPIKeys(userID)
}

// CreateAPIKey generates a server-side key bound to userID.
func (s *Service) CreateAPIKey(req CreateAPIKeyRequest) (CreatedAPIKey, error) {
	db, err := s.state()
	if err != nil {
		return CreatedAPIKey{}, err
	}
	userID := strings.TrimSpace(req.UserID)
	if userID == "" {
		return CreatedAPIKey{}, &ArgError{Field: "user_id", Message: "user_id is required"}
	}
	var expiresAt *time.Time
	if req.ExpiresInDays > 0 {
		t := time.Now().UTC().Add(time.Duration(req.ExpiresInDays) * 24 * time.Hour)
		expiresAt = &t
	}
	raw, entry, err := db.AddAPIKey(userID, req.Name, strings.Join(req.Scopes, ","), expiresAt)
	if err != nil {
		return CreatedAPIKey{}, err
	}
	return CreatedAPIKey{Raw: raw, Key: *entry}, nil
}

// PatchAPIKey updates an owned key and returns the redacted view.
func (s *Service) PatchAPIKey(id, userID string, patch store.APIKeyPatch) (store.APIKeyInfo, error) {
	db, err := s.state()
	if err != nil {
		return store.APIKeyInfo{}, err
	}
	if err := db.UpdateAPIKey(id, userID, patch); err != nil {
		return store.APIKeyInfo{}, err
	}
	key, err := db.GetAPIKey(id)
	if err != nil {
		return store.APIKeyInfo{}, err
	}
	return store.APIKeyInfo{
		ID: key.ID, UserID: key.UserID, Name: key.Name,
		Prefix: key.Prefix, Scopes: key.Scopes, Enabled: key.Enabled,
		ExpiresAt: key.ExpiresAt, LastUsedAt: key.LastUsedAt,
		CreatedAt: key.CreatedAt,
	}, nil
}

// DeleteAPIKey removes an owned key.
func (s *Service) DeleteAPIKey(id, userID string) error {
	db, err := s.state()
	if err != nil {
		return err
	}
	return db.DeleteAPIKey(id, userID)
}
