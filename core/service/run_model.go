package service

import (
	"strings"

	"github.com/teexue/nexa/core/agent"
	"github.com/teexue/nexakit/session"
)

func (s *Service) applyRunModel(req RunRequest, sess *session.Session, a *agent.Agent) error {
	providerName, model, err := s.resolveRunModel(req, sess, a)
	if err != nil {
		return err
	}
	a.Provider = providerName
	a.Model = model
	sess.SetMetadata(session.MetadataKeyProvider, providerName)
	sess.SetMetadata(session.MetadataKeyModel, model)
	return nil
}

func (s *Service) resolveRunModel(req RunRequest, sess *session.Session, a *agent.Agent) (string, string, error) {
	meta := sess.GetMetadata()
	if locked := strings.TrimSpace(meta[session.MetadataKeyModel]); locked != "" {
		return lockedSessionModel(req, meta, a, locked)
	}
	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = a.Model
	}
	providerName := strings.TrimSpace(req.Provider)
	if providerName == "" {
		providerName = s.providerForModel(model, a.Provider)
	}
	if err := s.ensureModelAllowed(providerName, model); err != nil {
		return "", "", err
	}
	return providerName, model, nil
}

func lockedSessionModel(req RunRequest, meta map[string]string, a *agent.Agent, locked string) (string, string, error) {
	want := strings.TrimSpace(req.Model)
	if want != "" && want != locked {
		return "", "", &ArgError{Field: "model", Message: "session model is locked"}
	}
	providerName := strings.TrimSpace(meta[session.MetadataKeyProvider])
	if providerName == "" {
		providerName = a.Provider
	}
	wantProvider := strings.TrimSpace(req.Provider)
	if wantProvider != "" && wantProvider != providerName {
		return "", "", &ArgError{Field: "provider", Message: "session provider is locked"}
	}
	return providerName, locked, nil
}

func (s *Service) providerForModel(model, fallback string) string {
	if s.Catalog == nil {
		return fallback
	}
	if fallback != "" && s.Catalog.ModelAllowed(fallback, model) {
		return fallback
	}
	if name, ok := s.Catalog.ProviderForModel(model); ok {
		return name
	}
	return fallback
}

func (s *Service) ensureModelAllowed(providerName, model string) error {
	if s.Catalog == nil {
		return nil
	}
	if s.Catalog.ModelAllowed(providerName, model) {
		return nil
	}
	return &ArgError{Field: "model", Message: "model is not in the provider's enabled list"}
}
