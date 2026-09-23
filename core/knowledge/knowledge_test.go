package knowledge_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	embmock "github.com/teexue/nexakit/embedding/mock"

	"github.com/teexue/nexa/core/knowledge"
)

func TestChunk(t *testing.T) {
	parts := knowledge.ChunkSized("hello\n\nworld", 5, 0)
	require.Len(t, parts, 2)
	assert.Equal(t, "hello", parts[0])
	assert.Equal(t, "world", parts[1])

	merged := knowledge.Chunk("hello\n\nworld")
	require.Len(t, merged, 1)
	assert.Contains(t, merged[0], "hello")
}

func TestKnowledgeRoundTrip(t *testing.T) {
	root := t.TempDir()
	mgr, err := knowledge.NewManager(root)
	require.NoError(t, err)

	meta, err := mgr.Create("product_docs", "Product Docs", "demo")
	require.NoError(t, err)
	assert.Equal(t, "product_docs", meta.ID)

	emb := &embmock.MockEmbedder{Dim: 8}
	ing := knowledge.NewIngester(mgr, emb)
	ret := knowledge.NewRetriever(mgr, emb)

	doc, err := ing.AddDocument(context.Background(), "product_docs", "guide.md", []byte("# Guide\n\nCommon agent uses tools for RAG retrieval.\n\nKnowledge bases store embeddings."))
	require.NoError(t, err)
	assert.NotEmpty(t, doc.ID)

	docs, err := ing.ListDocuments("product_docs")
	require.NoError(t, err)
	require.Len(t, docs, 1)

	got, err := mgr.Get("product_docs")
	require.NoError(t, err)
	assert.Equal(t, 1, got.DocCount)
	assert.Greater(t, got.ChunkCount, 0)

	hits, err := ret.Search(context.Background(), knowledge.SearchOptions{
		Query: "RAG retrieval tools",
		KBIDs: []string{"product_docs"},
		TopK:  3,
	})
	require.NoError(t, err)
	require.NotEmpty(t, hits)
	assert.Equal(t, "product_docs", hits[0].KBID)

	require.NoError(t, ing.DeleteDocument("product_docs", doc.ID))
	docs, err = ing.ListDocuments("product_docs")
	require.NoError(t, err)
	assert.Empty(t, docs)

	require.NoError(t, mgr.Delete("product_docs"))
	_, err = mgr.Get("product_docs")
	assert.Error(t, err)
}

func TestValidateID(t *testing.T) {
	assert.Error(t, knowledge.ValidateID("../x"))
	assert.NoError(t, knowledge.ValidateID("kb_demo"))
}
