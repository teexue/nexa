package service

import (
	"strings"

	"github.com/teexue/nexa/core/store"
)

// ListUsers returns all users as public views.
func (s *Service) ListUsers() ([]store.UserInfo, error) {
	db, err := s.state()
	if err != nil {
		return nil, err
	}
	users, err := db.ListUsers()
	if err != nil {
		return nil, err
	}
	infos := make([]store.UserInfo, 0, len(users))
	for _, u := range users {
		infos = append(infos, u.ToUserInfo())
	}
	return infos, nil
}

// AdminCreateUser creates a user with an explicit role.
func (s *Service) AdminCreateUser(username, password, name, role string) (store.UserInfo, error) {
	db, err := s.state()
	if err != nil {
		return store.UserInfo{}, err
	}
	u, err := db.CreateUser(username, password, name, role)
	if err != nil {
		return store.UserInfo{}, err
	}
	return u.ToUserInfo(), nil
}

// AdminPatchUser changes role and/or password, then returns the updated view.
func (s *Service) AdminPatchUser(id, role, password string) (store.UserInfo, error) {
	db, err := s.state()
	if err != nil {
		return store.UserInfo{}, err
	}
	if role = strings.TrimSpace(role); role != "" {
		if err := db.UpdateUserRole(id, role); err != nil {
			return store.UserInfo{}, err
		}
	}
	if password != "" {
		if err := db.ResetUserPassword(id, password); err != nil {
			return store.UserInfo{}, err
		}
	}
	u, err := db.GetUser(id)
	if err != nil {
		return store.UserInfo{}, err
	}
	return u.ToUserInfo(), nil
}

// AdminDeleteUser removes a user and their API keys.
func (s *Service) AdminDeleteUser(id string) error {
	db, err := s.state()
	if err != nil {
		return err
	}
	return db.DeleteUser(id)
}

// AllowRegistration reports the open-registration setting.
func (s *Service) AllowRegistration() (bool, error) {
	db, err := s.state()
	if err != nil {
		return false, err
	}
	return db.GetAllowRegistration(), nil
}

// SetAllowRegistration toggles open self-registration and returns the stored value.
func (s *Service) SetAllowRegistration(allow bool) (bool, error) {
	db, err := s.state()
	if err != nil {
		return false, err
	}
	if err := db.SetAllowRegistration(allow); err != nil {
		return false, err
	}
	return db.GetAllowRegistration(), nil
}
