package service

import (
	"context"
	"fmt"
	"os"

	"github.com/teexue/common-agent/core/config"
	"github.com/teexue/nexakit/embedding"
	"github.com/teexue/common-agent/core/knowledge"
)

// CreateKnowledge creates a knowledge base.
func (s *Service) CreateKnowledge(id, name, description string) (*knowledge.Meta, error) {
	if s.Knowledge == nil {
		return nil, &ServerError{Message: "knowledge is not configured"}
	}
	return s.Knowledge.Create(id, name, description)
}

// ListKnowledge returns all knowledge bases.
func (s *Service) ListKnowledge() ([]knowledge.Meta, error) {
	if s.Knowledge == nil {
		return nil, &ServerError{Message: "knowledge is not configured"}
	}
	return s.Knowledge.List()
}

// GetKnowledge returns one knowledge base.
func (s *Service) GetKnowledge(id string) (*knowledge.Meta, error) {
	if s.Knowledge == nil {
		return nil, &ServerError{Message: "knowledge is not configured"}
	}
	return s.Knowledge.Get(id)
}

// UpdateKnowledge updates name/description.
func (s *Service) UpdateKnowledge(id, name, description string) (*knowledge.Meta, error) {
	if s.Knowledge == nil {
		return nil, &ServerError{Message: "knowledge is not configured"}
	}
	return s.Knowledge.Update(id, name, description)
}

// DeleteKnowledge removes a knowledge base.
func (s *Service) DeleteKnowledge(id string) error {
	if s.Knowledge == nil {
		return &ServerError{Message: "knowledge is not configured"}
	}
	return s.Knowledge.Delete(id)
}

// AddKnowledgeDocument ingests a document.
func (s *Service) AddKnowledgeDocument(ctx context.Context, kbID, filename string, content []byte) (*knowledge.Document, error) {
	ing := s.Ingester
	if s.KnowledgeRuntime != nil {
		ing = s.KnowledgeRuntime.CurrentIngester()
	}
	if ing == nil {
		return nil, &ServerError{Message: "knowledge is not configured"}
	}
	emb := s.Embedder
	if s.KnowledgeRuntime != nil {
		emb = s.KnowledgeRuntime.CurrentEmbedder()
	}
	if emb == nil {
		return nil, &ArgError{Field: "embedding", Message: "embedding model is not configured; set embedding in Settings"}
	}
	return ing.AddDocument(ctx, kbID, filename, content)
}

// DeleteKnowledgeDocument removes a document.
func (s *Service) DeleteKnowledgeDocument(kbID, docID string) error {
	ing := s.Ingester
	if s.KnowledgeRuntime != nil {
		ing = s.KnowledgeRuntime.CurrentIngester()
	}
	if ing == nil {
		return &ServerError{Message: "knowledge is not configured"}
	}
	return ing.DeleteDocument(kbID, docID)
}

// ListKnowledgeDocuments lists documents in a KB.
func (s *Service) ListKnowledgeDocuments(kbID string) ([]knowledge.Document, error) {
	ing := s.Ingester
	if s.KnowledgeRuntime != nil {
		ing = s.KnowledgeRuntime.CurrentIngester()
	}
	if ing == nil {
		return nil, &ServerError{Message: "knowledge is not configured"}
	}
	return ing.ListDocuments(kbID)
}

// ReindexKnowledge rebuilds embeddings for a KB.
func (s *Service) ReindexKnowledge(ctx context.Context, kbID string) error {
	ing := s.Ingester
	if s.KnowledgeRuntime != nil {
		ing = s.KnowledgeRuntime.CurrentIngester()
	}
	if ing == nil {
		return &ServerError{Message: "knowledge is not configured"}
	}
	emb := s.Embedder
	if s.KnowledgeRuntime != nil {
		emb = s.KnowledgeRuntime.CurrentEmbedder()
	}
	if emb == nil {
		return &ArgError{Field: "embedding", Message: "embedding model is not configured; set embedding in Settings"}
	}
	return ing.Reindex(ctx, kbID)
}

// SearchKnowledge runs a debug/admin search.
func (s *Service) SearchKnowledge(ctx context.Context, query string, kbIDs []string, topK int) ([]knowledge.Hit, error) {
	if s.KnowledgeRuntime != nil {
		if s.KnowledgeRuntime.CurrentEmbedder() == nil {
			return nil, &ArgError{Field: "embedding", Message: "embedding model is not configured; set embedding in Settings"}
		}
		return s.KnowledgeRuntime.Search(ctx, knowledge.SearchOptions{Query: query, KBIDs: kbIDs, TopK: topK})
	}
	if s.Retriever == nil {
		return nil, &ServerError{Message: "knowledge is not configured"}
	}
	if s.Embedder == nil {
		return nil, &ArgError{Field: "embedding", Message: "embedding model is not configured; set embedding in Settings"}
	}
	return s.Retriever.Search(ctx, knowledge.SearchOptions{Query: query, KBIDs: kbIDs, TopK: topK})
}

// GetEmbeddingSettings returns the embedding section of user settings.
func (s *Service) GetEmbeddingSettings() (embedding.ConfigView, error) {
	home := s.HomeDir
	if home == "" {
		return embedding.ConfigView{}, fmt.Errorf("home dir not set")
	}
	settings, err := config.LoadSettings(home)
	if err != nil {
		return embedding.ConfigView{}, err
	}
	cfg := embedding.Config{Backend: embedding.BackendOpenAI}
	if settings.Embedding != nil {
		cfg = settings.Embedding.Normalize()
	}
	hasKey := false
	if cfg.APIKeyEnv != "" {
		if v := os.Getenv(cfg.APIKeyEnv); v != "" {
			hasKey = true
		} else if s.Creds != nil && s.Creds.Get(cfg.APIKeyEnv) != "" {
			hasKey = true
		}
	}
	return cfg.View(hasKey), nil
}

// SaveEmbeddingRequest is the PUT /v1/embedding body.
type SaveEmbeddingRequest struct {
	Vendor     string `json:"vendor"`
	Backend    string `json:"backend"`
	BaseURL    string `json:"base_url"`
	APIKeyEnv  string `json:"api_key_env"`
	Model      string `json:"model"`
	Dimensions int    `json:"dimensions"`
	APIKey     string `json:"api_key,omitempty"`
}

// SaveEmbeddingSettings persists embedding config, optionally stores api_key,
// and rebuilds the embedder.
func (s *Service) SaveEmbeddingSettings(req SaveEmbeddingRequest) error {
	home := s.HomeDir
	if home == "" {
		return fmt.Errorf("home dir not set")
	}
	cfg := embedding.Config{
		Vendor:     req.Vendor,
		Backend:    req.Backend,
		BaseURL:    req.BaseURL,
		APIKeyEnv:  req.APIKeyEnv,
		Model:      req.Model,
		Dimensions: req.Dimensions,
	}.Normalize()
	if err := s.storeEmbeddingKey(cfg, req.APIKey); err != nil {
		return err
	}
	if err := validateEmbeddingCfg(cfg); err != nil {
		return err
	}
	settings, err := config.LoadSettings(home)
	if err != nil {
		return err
	}
	persisted := cfg.Persistable()
	settings.Embedding = &persisted
	if err := config.SaveSettings(home, settings); err != nil {
		return err
	}
	return s.applyEmbedder(cfg)
}

func (s *Service) storeEmbeddingKey(cfg embedding.Config, apiKey string) error {
	if apiKey == "" {
		return nil
	}
	if cfg.APIKeyEnv == "" {
		return &ArgError{Field: "api_key_env", Message: "api_key_env is required when setting api_key"}
	}
	if s.Creds == nil {
		return &ServerError{Message: "credential store is not configured"}
	}
	if err := s.Creds.Set(cfg.APIKeyEnv, apiKey); err != nil {
		return fmt.Errorf("store api key: %w", err)
	}
	return nil
}

func validateEmbeddingCfg(cfg embedding.Config) error {
	if cfg.Model == "" {
		return &ArgError{Field: "model", Message: "embedding.model is required"}
	}
	if cfg.Backend == embedding.BackendOpenAI && cfg.BaseURL == "" {
		return &ArgError{Field: "base_url", Message: "embedding.base_url is required"}
	}
	if cfg.Backend == embedding.BackendOpenAI && cfg.APIKeyEnv == "" {
		return &ArgError{Field: "api_key_env", Message: "embedding.api_key_env is required"}
	}
	return nil
}

func (s *Service) applyEmbedder(cfg embedding.Config) error {
	lookup := embedding.KeyLookup(os.Getenv)
	if s.Creds != nil {
		lookup = s.Creds.Lookup
	}
	emb, err := embedding.New(cfg, lookup)
	if err != nil {
		return &ArgError{Field: "embedding", Message: err.Error()}
	}
	s.Embedder = emb
	if s.Knowledge == nil {
		return nil
	}
	s.Ingester = knowledge.NewIngester(s.Knowledge, emb)
	s.Retriever = knowledge.NewRetriever(s.Knowledge, emb)
	if s.KnowledgeRuntime != nil {
		s.KnowledgeRuntime.SetEmbedder(emb)
	}
	return nil
}

// ListEmbeddingVendors returns built-in embedding vendor presets.
func (s *Service) ListEmbeddingVendors() []embedding.VendorInfo {
	return embedding.ListVendors()
}
