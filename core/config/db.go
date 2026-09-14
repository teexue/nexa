package config

import (
	"errors"

	"github.com/teexue/nexa/core/store"
)

// ErrDBNotBound is returned when config helpers run before BindDB / OpenAndBind.
var ErrDBNotBound = errors.New("state.db is not open")

// stateDB is the process-wide SQLite store after BindDB.
var stateDB *store.DB

// BindDB sets the shared SQLite store used by config helpers.
func BindDB(db *store.DB) { stateDB = db }

// DB returns the bound store, or nil.
func DB() *store.DB { return stateDB }

// OpenAndBind opens state.db under home and BindDB. The caller must Close the
// DB (and typically BindDB(nil) in tests).
func OpenAndBind(home string) (*store.DB, error) {
	db, err := store.Open(home)
	if err != nil {
		return nil, err
	}
	BindDB(db)
	return db, nil
}

func requireDB() (*store.DB, error) {
	if stateDB == nil {
		return nil, ErrDBNotBound
	}
	return stateDB, nil
}
