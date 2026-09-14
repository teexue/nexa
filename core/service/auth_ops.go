package service

import (
	"errors"

	"github.com/teexue/nexa/core/store"
)

// ErrStateNotConfigured is returned when auth/admin ops run without a state DB.
var ErrStateNotConfigured = errors.New("state.db is not configured")

// ErrRegistrationDisabled is returned when self-registration is off and users exist.
var ErrRegistrationDisabled = errors.New("registration is disabled")

func (s *Service) state() (*store.DB, error) {
	if s == nil || s.StateDB == nil {
		return nil, ErrStateNotConfigured
	}
	return s.StateDB, nil
}

// AuthStatus is the public probe for the login gate.
type AuthStatus struct {
	HasUsers          bool
	AllowRegistration bool
}

// AuthProbe reports whether password users exist and whether registration is open.
func (s *Service) AuthProbe() AuthStatus {
	db, err := s.state()
	if err != nil {
		return AuthStatus{}
	}
	n, err := db.CountUsersWithPassword()
	return AuthStatus{
		HasUsers:          err == nil && n > 0,
		AllowRegistration: db.GetAllowRegistration(),
	}
}

// RegisterUser creates a password user. The first password user is admin;
// later registrations require the allow_registration flag and are members.
func (s *Service) RegisterUser(username, password, name string) (*store.User, error) {
	db, err := s.state()
	if err != nil {
		return nil, err
	}
	role := store.RoleMember
	n, err := db.CountUsersWithPassword()
	if err != nil {
		return nil, &ServerError{Message: err.Error()}
	}
	if n == 0 {
		role = store.RoleAdmin
	} else if !db.GetAllowRegistration() {
		return nil, ErrRegistrationDisabled
	}
	return db.CreateUser(username, password, name, role)
}

// AuthenticateUser verifies username and password.
func (s *Service) AuthenticateUser(username, password string) (store.User, error) {
	db, err := s.state()
	if err != nil {
		return store.User{}, err
	}
	return db.AuthenticateUser(username, password)
}

// LookupUser returns a public user view when the id exists.
func (s *Service) LookupUser(id string) (store.UserInfo, bool) {
	db, err := s.state()
	if err != nil || id == "" {
		return store.UserInfo{}, false
	}
	u, err := db.GetUser(id)
	if err != nil {
		return store.UserInfo{}, false
	}
	return u.ToUserInfo(), true
}

// VerifyStoredAPIKey checks a raw key against persisted hashes.
// A nil key with a nil error means the key is not in the store.
func (s *Service) VerifyStoredAPIKey(raw string) (*store.APIKey, error) {
	db, err := s.state()
	if err != nil {
		return nil, err
	}
	return db.VerifyAPIKey(raw)
}
