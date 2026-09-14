package httpapi

import (
	"context"
	"path/filepath"

	"github.com/teexue/nexa/core/config"
	"github.com/teexue/nexakit/provider"
)

// modelWindowsForUpsert builds the model→window map to persist with a provider.
// The request's context_window (from the UI's /api/show result) wins; otherwise
// an Ollama provider is queried live for the default model.
func (s *Server) modelWindowsForUpsert(ctx context.Context, req ProviderUpsertRequest) map[string]int {
	windows := map[string]int{}
	if req.DefaultModel != "" && req.ContextWindow > 0 {
		windows[req.DefaultModel] = req.ContextWindow
	}
	if len(windows) > 0 {
		return windows
	}
	if n := s.ollamaContextWindow(ctx, req); n > 0 {
		return map[string]int{req.DefaultModel: n}
	}
	return nil
}

func (s *Server) ollamaContextWindow(ctx context.Context, req ProviderUpsertRequest) int {
	if provider.APIStyle(req.APIStyle) != provider.StyleOllama || req.DefaultModel == "" {
		return 0
	}
	p, err := s.buildInlineProvider(ProviderModelsRequest{
		Name:       req.Name,
		APIStyle:   req.APIStyle,
		BaseURL:    req.BaseURL,
		ModelsPath: req.ModelsPath,
		APIVersion: req.APIVersion,
		AuthStyle:  req.AuthStyle,
		APIKey:     req.APIKey,
	})
	if err != nil {
		return 0
	}
	d, ok := p.(provider.ModelDetailer)
	if !ok {
		return 0
	}
	detail, err := d.ShowModel(ctx, req.DefaultModel)
	if err != nil || detail.ContextWindow <= 0 {
		return 0
	}
	return detail.ContextWindow
}

func (s *Server) rememberModelWindow(name, model string, window int) {
	if window <= 0 || name == "" || model == "" {
		return
	}
	home := s.home
	if home == "" {
		home = filepath.Dir(s.agentsDir)
	}
	changed, err := config.MergeProviderModelWindow(home, name, model, window)
	if err != nil {
		if s.logger != nil {
			s.logger.Warn("remember model window", "provider", name, "model", model, "error", err)
		}
		return
	}
	if changed {
		_ = s.reloadCatalog()
	}
}
