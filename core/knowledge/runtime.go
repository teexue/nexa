package knowledge

import (
	"context"
	"sync"

	"github.com/teexue/nexakit/embedding"
)

// Runtime is a shared, swappable knowledge stack used by tools and HTTP.
type Runtime struct {
	mu        sync.RWMutex
	Manager   *Manager
	Embedder  embedding.Embedder
	Ingester  *Ingester
	Retriever *Retriever
}

// NewRuntime builds a Runtime around mgr and emb (emb may be nil until configured).
func NewRuntime(mgr *Manager, emb embedding.Embedder) *Runtime {
	r := &Runtime{Manager: mgr, Embedder: emb}
	r.Ingester = NewIngester(mgr, emb)
	r.Retriever = NewRetriever(mgr, emb)
	return r
}

// SetEmbedder hot-swaps the embedding backend and rebuilds ingest/retrieve helpers.
func (r *Runtime) SetEmbedder(emb embedding.Embedder) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Embedder = emb
	r.Ingester = NewIngester(r.Manager, emb)
	r.Retriever = NewRetriever(r.Manager, emb)
}

// Search delegates to the current retriever.
func (r *Runtime) Search(ctx context.Context, opts SearchOptions) ([]Hit, error) {
	r.mu.RLock()
	ret := r.Retriever
	r.mu.RUnlock()
	return ret.Search(ctx, opts)
}

// ListBases lists knowledge bases.
func (r *Runtime) ListBases() ([]Meta, error) {
	r.mu.RLock()
	mgr := r.Manager
	r.mu.RUnlock()
	return mgr.List()
}

// CurrentIngester returns the active ingester.
func (r *Runtime) CurrentIngester() *Ingester {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.Ingester
}

// CurrentRetriever returns the active retriever.
func (r *Runtime) CurrentRetriever() *Retriever {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.Retriever
}

// CurrentEmbedder returns the active embedder (may be nil).
func (r *Runtime) CurrentEmbedder() embedding.Embedder {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.Embedder
}
