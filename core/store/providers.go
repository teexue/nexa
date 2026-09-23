package store

import (
	"encoding/json"
	"fmt"

	kitcatalog "github.com/teexue/nexakit/provider/catalog"
)

// ListProviderEntries returns all provider profile entries.
func (db *DB) ListProviderEntries() (map[string]kitcatalog.ProfileEntry, error) {
	var rows []ProviderRow
	if err := db.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[string]kitcatalog.ProfileEntry, len(rows))
	for _, r := range rows {
		var e kitcatalog.ProfileEntry
		if err := json.Unmarshal([]byte(r.SpecJSON), &e); err != nil {
			return nil, fmt.Errorf("parse provider %q: %w", r.Name, err)
		}
		out[r.Name] = e
	}
	return out, nil
}

// UpsertProviderEntry saves a provider profile entry.
func (db *DB) UpsertProviderEntry(name string, entry kitcatalog.ProfileEntry) error {
	if name == "" {
		return fmt.Errorf("provider name is required")
	}
	b, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal provider: %w", err)
	}
	return db.Save(&ProviderRow{Name: name, SpecJSON: string(b)}).Error
}

// DeleteProviderEntry removes a provider by name.
func (db *DB) DeleteProviderEntry(name string) error {
	if name == "" {
		return fmt.Errorf("provider name is required")
	}
	res := db.Where("name = ?", name).Delete(&ProviderRow{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("provider %q not found", name)
	}
	return nil
}

// LoadCatalog builds a kitcatalog.Catalog from the DB.
func (db *DB) LoadCatalog(credLookup func(string) string) (*kitcatalog.Catalog, error) {
	entries, err := db.ListProviderEntries()
	if err != nil {
		return nil, err
	}
	return kitcatalog.NewCatalog(entries, credLookup)
}
