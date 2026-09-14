package knowledge

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/teexue/nexakit/registry"
	"github.com/teexue/nexakit/tool"
)

// KnowledgeSearch retrieves relevant fragments from configured knowledge bases.
type KnowledgeSearch struct {
	Runtime  *Runtime
	Allowed  []string
	DefaultK int
}

// Name returns the tool name.
func (KnowledgeSearch) Name() string { return "knowledge_search" }

// Description returns a human-readable description.
func (KnowledgeSearch) Description() string {
	return "Search the knowledge base for relevant document fragments. Use when answering questions that may be covered by uploaded docs."
}

// InputSchema returns the JSON Schema for the tool's input.
func (KnowledgeSearch) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{"type": "string", "description": "Natural language search query"},
			"kb_id": map[string]any{"type": "string", "description": "Optional knowledge base id to search"},
			"top_k": map[string]any{"type": "integer", "description": "Max fragments to return (default 5)"},
		},
		"required": []string{"query"},
	}
}

// Execute runs the search.
func (t KnowledgeSearch) Execute(ctx context.Context, input json.RawMessage) (tool.Result, error) {
	if t.Runtime == nil {
		return tool.Result{}, fmt.Errorf("knowledge search is not configured")
	}
	args, err := parseSearchArgs(input)
	if err != nil {
		return tool.Result{}, err
	}
	allowed, topK := resolveScope(ctx, t.Allowed, t.DefaultK, args.TopK)
	if err := rejectDisallowedKB(allowed, args.KBID); err != nil {
		return tool.Result{}, err
	}
	kbIDs := allowed
	if args.KBID != "" {
		kbIDs = []string{args.KBID}
	}
	hits, err := t.Runtime.Search(ctx, SearchOptions{Query: args.Query, KBIDs: kbIDs, TopK: topK})
	if err != nil {
		return tool.Result{}, err
	}
	out, _ := json.Marshal(map[string]any{"hits": hits, "count": len(hits)})
	return tool.Result{Output: out}, nil
}

type searchArgs struct {
	Query string `json:"query"`
	KBID  string `json:"kb_id"`
	TopK  int    `json:"top_k"`
}

func parseSearchArgs(input json.RawMessage) (searchArgs, error) {
	var args searchArgs
	if err := json.Unmarshal(input, &args); err != nil {
		return args, fmt.Errorf("parse knowledge_search input: %w", err)
	}
	return args, nil
}

func resolveScope(ctx context.Context, allowed []string, defaultK, topK int) ([]string, int) {
	if topK <= 0 {
		topK = defaultK
	}
	if scope, ok := ScopeFrom(ctx); ok {
		if len(scope.Bases) > 0 {
			allowed = scope.Bases
		}
		if topK <= 0 && scope.TopK > 0 {
			topK = scope.TopK
		}
	}
	if topK <= 0 {
		topK = 5
	}
	return allowed, topK
}

func rejectDisallowedKB(allowed []string, kbID string) error {
	if kbID == "" || len(allowed) == 0 || containsStr(allowed, kbID) {
		return nil
	}
	return fmt.Errorf("knowledge base %q is not allowed for this agent", kbID)
}

// KnowledgeList lists available knowledge bases.
type KnowledgeList struct {
	Runtime *Runtime
	Allowed []string
}

// Name returns the tool name.
func (KnowledgeList) Name() string { return "knowledge_list" }

// Description returns a human-readable description.
func (KnowledgeList) Description() string {
	return "List available knowledge bases (id, name, document counts)."
}

// InputSchema returns the JSON Schema for the tool's input.
func (KnowledgeList) InputSchema() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}

// Execute lists knowledge bases.
func (t KnowledgeList) Execute(ctx context.Context, _ json.RawMessage) (tool.Result, error) {
	if t.Runtime == nil {
		return tool.Result{}, fmt.Errorf("knowledge list is not configured")
	}
	metas, err := t.Runtime.ListBases()
	if err != nil {
		return tool.Result{}, err
	}
	out, _ := json.Marshal(map[string]any{"bases": filterBases(ctx, t.Allowed, metas)})
	return tool.Result{Output: out}, nil
}

func filterBases(ctx context.Context, allowed []string, metas []Meta) []Meta {
	if scope, ok := ScopeFrom(ctx); ok && len(scope.Bases) > 0 {
		allowed = scope.Bases
	}
	if len(allowed) == 0 {
		return metas
	}
	filtered := metas[:0]
	for _, m := range metas {
		if containsStr(allowed, m.ID) {
			filtered = append(filtered, m)
		}
	}
	return filtered
}

// RegisterKnowledge registers knowledge tools when runtime is non-nil.
func RegisterKnowledge(r *registry.Registry, rt *Runtime) {
	if rt == nil {
		return
	}
	r.MustRegister(KnowledgeSearch{Runtime: rt, DefaultK: 5})
	r.MustRegister(KnowledgeList{Runtime: rt})
}

func containsStr(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}
