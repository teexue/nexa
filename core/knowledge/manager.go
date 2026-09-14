// Package knowledge provides document knowledge bases with vector retrieval.
package knowledge

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"gopkg.in/yaml.v3"
)

var idPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// Meta describes a knowledge base.
type Meta struct {
	ID          string    `yaml:"id" json:"id"`
	Name        string    `yaml:"name" json:"name"`
	Description string    `yaml:"description,omitempty" json:"description,omitempty"`
	CreatedAt   time.Time `yaml:"created_at" json:"created_at"`
	UpdatedAt   time.Time `yaml:"updated_at" json:"updated_at"`
	DocCount    int       `yaml:"-" json:"doc_count"`
	ChunkCount  int       `yaml:"-" json:"chunk_count"`
}

// Document describes an ingested file.
type Document struct {
	ID         string    `json:"id"`
	Filename   string    `json:"filename"`
	Size       int64     `json:"size"`
	CreatedAt  time.Time `json:"created_at"`
	ChunkCount int       `json:"chunk_count"`
}

// Hit is a retrieval result fragment.
type Hit struct {
	KBID       string  `json:"kb_id"`
	DocID      string  `json:"doc_id"`
	Filename   string  `json:"filename"`
	ChunkIndex int     `json:"chunk_index"`
	Text       string  `json:"text"`
	Score      float32 `json:"score"`
}

// ValidateID checks knowledge base / document id format.
func ValidateID(id string) error {
	if id == "" {
		return fmt.Errorf("id is required")
	}
	if !idPattern.MatchString(id) {
		return fmt.Errorf("invalid id %q: must match %s", id, idPattern.String())
	}
	return nil
}

// Manager owns knowledge bases under a root directory.
type Manager struct {
	root string
}

// NewManager creates a Manager rooted at dir (typically ~/.nexa/knowledge).
func NewManager(root string) (*Manager, error) {
	if root == "" {
		return nil, fmt.Errorf("knowledge root is required")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("create knowledge root: %w", err)
	}
	return &Manager{root: root}, nil
}

// Root returns the knowledge root directory.
func (m *Manager) Root() string { return m.root }

func (m *Manager) kbDir(id string) string {
	return filepath.Join(m.root, id)
}

func (m *Manager) metaPath(id string) string {
	return filepath.Join(m.kbDir(id), "meta.yaml")
}

func (m *Manager) docsDir(id string) string {
	return filepath.Join(m.kbDir(id), "docs")
}

func (m *Manager) indexPath(id string) string {
	return filepath.Join(m.kbDir(id), "index.db")
}

// Create creates a new knowledge base.
func (m *Manager) Create(id, name, description string) (*Meta, error) {
	if err := ValidateID(id); err != nil {
		return nil, err
	}
	if name == "" {
		name = id
	}
	dir := m.kbDir(id)
	if _, err := os.Stat(dir); err == nil {
		return nil, fmt.Errorf("knowledge base %q already exists", id)
	}
	if err := os.MkdirAll(m.docsDir(id), 0o755); err != nil {
		return nil, fmt.Errorf("create kb dirs: %w", err)
	}
	now := time.Now().UTC()
	meta := Meta{
		ID:          id,
		Name:        name,
		Description: description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := m.writeMeta(meta); err != nil {
		_ = os.RemoveAll(dir)
		return nil, err
	}
	idx, err := openIndex(m.indexPath(id))
	if err != nil {
		_ = os.RemoveAll(dir)
		return nil, err
	}
	_ = idx.Close()
	return &meta, nil
}

// Get returns knowledge base metadata with counts.
func (m *Manager) Get(id string) (*Meta, error) {
	meta, err := m.readMeta(id)
	if err != nil {
		return nil, err
	}
	idx, err := openIndex(m.indexPath(id))
	if err != nil {
		return meta, nil
	}
	defer idx.Close()
	meta.DocCount, meta.ChunkCount, _ = idx.counts()
	return meta, nil
}

// List returns all knowledge bases.
func (m *Manager) List() ([]Meta, error) {
	entries, err := os.ReadDir(m.root)
	if err != nil {
		return nil, fmt.Errorf("read knowledge root: %w", err)
	}
	var out []Meta
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		meta, err := m.Get(e.Name())
		if err != nil {
			continue
		}
		out = append(out, *meta)
	}
	return out, nil
}

// Update patches name/description.
func (m *Manager) Update(id, name, description string) (*Meta, error) {
	meta, err := m.readMeta(id)
	if err != nil {
		return nil, err
	}
	if name != "" {
		meta.Name = name
	}
	meta.Description = description
	meta.UpdatedAt = time.Now().UTC()
	if err := m.writeMeta(*meta); err != nil {
		return nil, err
	}
	return m.Get(id)
}

// Delete removes a knowledge base and all files.
func (m *Manager) Delete(id string) error {
	if err := ValidateID(id); err != nil {
		return err
	}
	dir := m.kbDir(id)
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("knowledge base %q: %w", id, os.ErrNotExist)
		}
		return err
	}
	return os.RemoveAll(dir)
}

func (m *Manager) readMeta(id string) (*Meta, error) {
	if err := ValidateID(id); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(m.metaPath(id))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("knowledge base %q: %w", id, os.ErrNotExist)
		}
		return nil, fmt.Errorf("read meta: %w", err)
	}
	var meta Meta
	if err := yaml.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("parse meta: %w", err)
	}
	meta.ID = id
	return &meta, nil
}

func (m *Manager) writeMeta(meta Meta) error {
	data, err := yaml.Marshal(meta)
	if err != nil {
		return fmt.Errorf("marshal meta: %w", err)
	}
	return os.WriteFile(m.metaPath(meta.ID), data, 0o644)
}

func (m *Manager) touch(id string) error {
	meta, err := m.readMeta(id)
	if err != nil {
		return err
	}
	meta.UpdatedAt = time.Now().UTC()
	return m.writeMeta(*meta)
}
