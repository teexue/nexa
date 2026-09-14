package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// MigrateFromFiles imports legacy YAML/JSON files into SQLite once.
func (db *DB) MigrateFromFiles() error {
	if err := db.migrateSettingsFile(); err != nil {
		return err
	}
	if err := db.migrateCredentialsFile(); err != nil {
		return err
	}
	if err := db.migrateProvidersFile(); err != nil {
		return err
	}
	if err := db.migrateMCPFile(); err != nil {
		return err
	}
	if err := db.migrateAPIKeysFile(); err != nil {
		return err
	}
	if err := db.migrateSessionsDir(); err != nil {
		return err
	}
	return nil
}

func (db *DB) alreadyMigrated(flag string) (bool, error) {
	v, err := db.GetMeta(flag)
	return v == "1", err
}

func (db *DB) markMigrated(flag string) error {
	return db.SetMeta(flag, "1")
}

func (db *DB) migrateSettingsFile() error {
	done, err := db.alreadyMigrated(metaMigratedSettings)
	if err != nil || done {
		return err
	}
	path := filepath.Join(db.home, "config.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return db.markMigrated(metaMigratedSettings)
		}
		return err
	}
	var raw struct {
		DefaultAgent string         `yaml:"default_agent"`
		Locale       string         `yaml:"locale"`
		Embedding    *fileEmbedding `yaml:"embedding"`
	}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("parse config.yaml: %w", err)
	}
	s := Settings{DefaultAgent: raw.DefaultAgent, Locale: raw.Locale}
	if raw.Embedding != nil {
		cfg := raw.Embedding.toConfig()
		s.Embedding = &cfg
	}
	if err := db.SaveSettings(s); err != nil {
		return err
	}
	return db.markMigrated(metaMigratedSettings)
}

func (db *DB) migrateCredentialsFile() error {
	done, err := db.alreadyMigrated(metaMigratedCreds)
	if err != nil || done {
		return err
	}
	path := filepath.Join(db.home, "credentials.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return db.markMigrated(metaMigratedCreds)
		}
		return err
	}
	var creds map[string]string
	if err := yaml.Unmarshal(data, &creds); err != nil {
		return fmt.Errorf("parse credentials.yaml: %w", err)
	}
	for k, v := range creds {
		if k == "" || v == "" {
			continue
		}
		if err := db.Save(&Credential{EnvName: k, Value: v}).Error; err != nil {
			return err
		}
	}
	return db.markMigrated(metaMigratedCreds)
}

func (db *DB) migrateProvidersFile() error {
	done, err := db.alreadyMigrated(metaMigratedProviders)
	if err != nil || done {
		return err
	}
	path := filepath.Join(db.home, "providers.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return db.markMigrated(metaMigratedProviders)
		}
		return err
	}
	var file CatalogFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return fmt.Errorf("parse providers.yaml: %w", err)
	}
	for name, entry := range file.Entries() {
		if err := db.UpsertProviderEntry(name, entry); err != nil {
			return err
		}
	}
	return db.markMigrated(metaMigratedProviders)
}

func (db *DB) migrateMCPFile() error {
	done, err := db.alreadyMigrated(metaMigratedMCP)
	if err != nil || done {
		return err
	}
	path := filepath.Join(db.home, "mcp.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return db.markMigrated(metaMigratedMCP)
		}
		return err
	}
	var f struct {
		Servers []fileMCPServer `yaml:"servers"`
	}
	if err := yaml.Unmarshal(data, &f); err != nil {
		return fmt.Errorf("parse mcp.yaml: %w", err)
	}
	for _, srv := range f.Servers {
		if err := db.UpsertGlobalMCP(srv.toConfig()); err != nil {
			return err
		}
	}
	return db.markMigrated(metaMigratedMCP)
}

func (db *DB) migrateAPIKeysFile() error {
	done, err := db.alreadyMigrated(metaMigratedAPIKeys)
	if err != nil || done {
		return err
	}
	path := filepath.Join(db.home, "api_keys.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return db.markMigrated(metaMigratedAPIKeys)
		}
		return err
	}
	var f struct {
		Keys []struct {
			ID        string    `yaml:"id"`
			Name      string    `yaml:"name"`
			Key       string    `yaml:"key"`
			CreatedAt time.Time `yaml:"created_at"`
		} `yaml:"keys"`
	}
	if err := yaml.Unmarshal(data, &f); err != nil {
		return fmt.Errorf("parse api_keys.yaml: %w", err)
	}
	for _, k := range f.Keys {
		if k.Key == "" {
			continue
		}
		id := k.ID
		if id == "" {
			id, err = generateID("ak")
			if err != nil {
				return err
			}
		}
		created := k.CreatedAt
		if created.IsZero() {
			created = time.Now().UTC()
		}
		row := APIKey{
			ID: id, UserID: "", Name: k.Name,
			KeyHash: HashAPIKey(k.Key), Prefix: KeyPrefix(k.Key),
			Scopes: "*", Enabled: true,
			CreatedAt: created,
		}
		if row.Name == "" {
			row.Name = id
		}
		if err := db.Save(&row).Error; err != nil {
			return err
		}
	}
	return db.markMigrated(metaMigratedAPIKeys)
}

func (db *DB) migrateSessionsDir() error {
	done, err := db.alreadyMigrated(metaMigratedSessions)
	if err != nil || done {
		return err
	}
	dir := filepath.Join(db.home, "sessions")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return db.markMigrated(metaMigratedSessions)
		}
		return err
	}
	ss := NewSessionStore(db)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var sf struct {
			ID        string            `json:"id"`
			UserID    string            `json:"user_id"`
			Agent     string            `json:"agent"`
			Title     string            `json:"title"`
			Messages  json.RawMessage   `json:"messages"`
			Metadata  map[string]string `json:"metadata"`
			CreatedAt string            `json:"created_at"`
			UpdatedAt string            `json:"updated_at"`
		}
		if err := json.Unmarshal(data, &sf); err != nil {
			continue
		}
		created, _ := time.Parse(time.RFC3339Nano, sf.CreatedAt)
		if created.IsZero() {
			created, _ = time.Parse(time.RFC3339, sf.CreatedAt)
		}
		updated, _ := time.Parse(time.RFC3339Nano, sf.UpdatedAt)
		if updated.IsZero() {
			updated, _ = time.Parse(time.RFC3339, sf.UpdatedAt)
		}
		userID := sf.UserID
		msgJSON := "[]"
		if len(sf.Messages) > 0 {
			msgJSON = string(sf.Messages)
		}
		metaJSON := ""
		if len(sf.Metadata) > 0 {
			b, _ := json.Marshal(sf.Metadata)
			metaJSON = string(b)
		}
		row := SessionRow{
			ID: sf.ID, UserID: userID, Agent: sf.Agent, Title: sf.Title,
			MessagesJSON: msgJSON, MetadataJSON: metaJSON,
			CreatedAt: created, UpdatedAt: updated,
		}
		if err := db.Save(&row).Error; err != nil {
			return err
		}
		_ = ss // keep for interface clarity
	}
	return db.markMigrated(metaMigratedSessions)
}
