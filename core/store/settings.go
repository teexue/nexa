package store

import (
	"encoding/json"
	"fmt"

	"github.com/teexue/nexakit/embedding"
)

const (
	settingDefaultAgent = "default_agent"
	settingLocale       = "locale"
	settingEmbedding    = "embedding"
	settingSubagent     = "subagent"
	settingShell        = "shell"
)

// SubagentSettings is the persisted global sub-agent section.
type SubagentSettings struct {
	Enabled       *bool `json:"enabled,omitempty"`
	MaxTurns      int   `json:"max_turns,omitempty"`
	MaxDepth      int   `json:"max_depth,omitempty"`
	Timeout       int   `json:"timeout,omitempty"`
	MaxConcurrent int   `json:"max_concurrent,omitempty"`
}

// Settings mirrors config.Settings for persistence.
type Settings struct {
	DefaultAgent string
	Locale       string
	Embedding    *embedding.Config
	Subagent     *SubagentSettings
	Shell        string
}

// LoadSettings reads settings from the DB with defaults.
func (db *DB) LoadSettings() (Settings, error) {
	s := Settings{DefaultAgent: "chat-assistant", Locale: "zh-CN"}
	if v, err := db.getSetting(settingDefaultAgent); err != nil {
		return Settings{}, err
	} else if v != "" {
		s.DefaultAgent = v
	}
	if v, err := db.getSetting(settingLocale); err != nil {
		return Settings{}, err
	} else if v != "" {
		s.Locale = v
	}
	if v, err := db.getSetting(settingEmbedding); err != nil {
		return Settings{}, err
	} else if v != "" {
		var emb embedding.Config
		if err := json.Unmarshal([]byte(v), &emb); err != nil {
			return Settings{}, fmt.Errorf("parse embedding settings: %w", err)
		}
		n := emb.Normalize()
		s.Embedding = &n
	}
	if err := db.loadSubagent(&s); err != nil {
		return Settings{}, err
	}
	v, err := db.getSetting(settingShell)
	if err != nil {
		return Settings{}, err
	}
	s.Shell = v
	return s, nil
}

// SaveSettings writes settings to the DB.
func (db *DB) SaveSettings(s Settings) error {
	if s.DefaultAgent == "" {
		s.DefaultAgent = "chat-assistant"
	}
	if s.Locale == "" {
		s.Locale = "zh-CN"
	}
	if err := db.setSetting(settingDefaultAgent, s.DefaultAgent); err != nil {
		return err
	}
	if err := db.setSetting(settingLocale, s.Locale); err != nil {
		return err
	}
	var emb any
	if s.Embedding != nil {
		n := s.Embedding.Normalize()
		emb = n
	}
	if err := db.saveJSONSetting(settingEmbedding, emb); err != nil {
		return err
	}
	if err := db.setSetting(settingShell, s.Shell); err != nil {
		return err
	}
	return db.saveJSONSetting(settingSubagent, s.Subagent)
}

func (db *DB) loadSubagent(s *Settings) error {
	v, err := db.getSetting(settingSubagent)
	if err != nil {
		return err
	}
	if v == "" {
		return nil
	}
	var sub SubagentSettings
	if err := json.Unmarshal([]byte(v), &sub); err != nil {
		return fmt.Errorf("parse subagent settings: %w", err)
	}
	s.Subagent = &sub
	return nil
}

func (db *DB) saveJSONSetting(key string, val any) error {
	if val == nil {
		return db.Where("key = ?", key).Delete(&Setting{}).Error
	}
	b, err := json.Marshal(val)
	if err != nil {
		return fmt.Errorf("marshal %s settings: %w", key, err)
	}
	return db.setSetting(key, string(b))
}

func (db *DB) getSetting(key string) (string, error) {
	var row Setting
	err := db.Where("key = ?", key).First(&row).Error
	if err != nil {
		if isNotFound(err) {
			return "", nil
		}
		return "", err
	}
	return row.Value, nil
}

func (db *DB) setSetting(key, value string) error {
	return db.Save(&Setting{Key: key, Value: value}).Error
}
