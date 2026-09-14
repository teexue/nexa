package knowledge

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/teexue/nexakit/embedding"
)

// Retriever searches knowledge bases by embedding similarity.
type Retriever struct {
	mgr *Manager
	emb embedding.Embedder
}

// NewRetriever creates a Retriever.
func NewRetriever(mgr *Manager, emb embedding.Embedder) *Retriever {
	return &Retriever{mgr: mgr, emb: emb}
}

// SearchOptions controls retrieval.
type SearchOptions struct {
	Query string
	KBIDs []string // empty = all bases
	TopK  int
}

// Search embeds the query and returns top-k hits across selected bases.
func (r *Retriever) Search(ctx context.Context, opts SearchOptions) ([]Hit, error) {
	if r.emb == nil {
		return nil, fmt.Errorf("embedder is not configured")
	}
	q := strings.TrimSpace(opts.Query)
	if q == "" {
		return nil, fmt.Errorf("query is required")
	}
	topK := opts.TopK
	if topK <= 0 {
		topK = 5
	}
	kbIDs := opts.KBIDs
	if len(kbIDs) == 0 {
		metas, err := r.mgr.List()
		if err != nil {
			return nil, err
		}
		for _, m := range metas {
			kbIDs = append(kbIDs, m.ID)
		}
	}
	vecs, err := r.emb.Embed(ctx, []string{q})
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}
	if len(vecs) == 0 {
		return nil, fmt.Errorf("embed query: empty result")
	}
	queryVec := vecs[0]

	var hits []Hit
	for _, id := range kbIDs {
		if err := ValidateID(id); err != nil {
			continue
		}
		part, err := r.searchKB(id, queryVec)
		if err != nil {
			continue
		}
		hits = append(hits, part...)
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].Score > hits[j].Score })
	if len(hits) > topK {
		hits = hits[:topK]
	}
	return hits, nil
}

func (r *Retriever) searchKB(kbID string, queryVec []float32) ([]Hit, error) {
	idx, err := openIndex(r.mgr.indexPath(kbID))
	if err != nil {
		return nil, err
	}
	defer idx.Close()
	chunks, err := idx.allChunks()
	if err != nil {
		return nil, err
	}
	hits := make([]Hit, 0, len(chunks))
	for _, c := range chunks {
		score := cosine(queryVec, c.Embedding)
		hits = append(hits, Hit{
			KBID:       kbID,
			DocID:      c.DocID,
			Filename:   c.Filename,
			ChunkIndex: c.Index,
			Text:       c.Text,
			Score:      score,
		})
	}
	return hits, nil
}
