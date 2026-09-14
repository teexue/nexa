package config

import (
	"fmt"

	"github.com/teexue/nexakit/embedding"
	"github.com/teexue/nexa/core/store"
)

// Settings holds user-level defaults persisted in state.db.
type Settings struct {
	DefaultAgent string
	Locale       string
	Embedding    *embedding.Config
	Subagent     *SubagentSettings
	Shell        string
}

// LoadSettings reads settings from SQLite.
func LoadSettings(home string) (Settings, error) {
	_ = home
	db, err := requireDB()
	if err != nil {
		return Settings{}, err
	}
	s, err := db.LoadSettings()
	if err != nil {
		return Settings{}, err
	}
	return Settings{
		DefaultAgent: s.DefaultAgent, Locale: s.Locale,
		Embedding: s.Embedding, Subagent: fromStoreSubagent(s.Subagent),
		Shell: s.Shell,
	}, nil
}

// SaveSettings writes settings to SQLite.
func SaveSettings(home string, s Settings) error {
	if err := ensureHome(home); err != nil {
		return err
	}
	db, err := requireDB()
	if err != nil {
		return err
	}
	return db.SaveSettings(store.Settings{
		DefaultAgent: s.DefaultAgent,
		Locale:       s.Locale,
		Embedding:    s.Embedding,
		Subagent:     toStoreSubagent(s.Subagent),
		Shell:        s.Shell,
	})
}

func ensureHome(home string) error {
	if err := EnsureDirs(home); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	return nil
}
