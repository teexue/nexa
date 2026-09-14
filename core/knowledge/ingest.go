package knowledge

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/teexue/nexakit/embedding"
)

var allowedExts = map[string]bool{
	".md":       true,
	".txt":      true,
	".markdown": true,
}

// Ingester adds documents to a knowledge base and embeds chunks.
type Ingester struct {
	mgr *Manager
	emb embedding.Embedder
}

// NewIngester creates an Ingester.
func NewIngester(mgr *Manager, emb embedding.Embedder) *Ingester {
	return &Ingester{mgr: mgr, emb: emb}
}

// AddDocument writes content to disk, chunks, embeds, and indexes it.
func (ing *Ingester) AddDocument(ctx context.Context, kbID, filename string, content []byte) (*Document, error) {
	if err := ValidateID(kbID); err != nil {
		return nil, err
	}
	if _, err := ing.mgr.readMeta(kbID); err != nil {
		return nil, err
	}
	ext := strings.ToLower(filepath.Ext(filename))
	if !allowedExts[ext] {
		return nil, fmt.Errorf("unsupported file type %q (allowed: .md .txt .markdown)", ext)
	}
	if len(content) == 0 {
		return nil, fmt.Errorf("document content is empty")
	}
	docID := newDocID()
	safeName := sanitizeFilename(filename)
	dst := filepath.Join(ing.mgr.docsDir(kbID), docID+"_"+safeName)
	if err := os.WriteFile(dst, content, 0o644); err != nil {
		return nil, fmt.Errorf("write document: %w", err)
	}
	doc := Document{
		ID:        docID,
		Filename:  safeName,
		Size:      int64(len(content)),
		CreatedAt: time.Now().UTC(),
	}
	if err := ing.indexDocument(ctx, kbID, doc, string(content)); err != nil {
		_ = os.Remove(dst)
		return nil, err
	}
	_ = ing.mgr.touch(kbID)
	idx, err := openIndex(ing.mgr.indexPath(kbID))
	if err == nil {
		defer idx.Close()
		docs, _ := idx.listDocuments()
		for _, d := range docs {
			if d.ID == docID {
				return &d, nil
			}
		}
	}
	return &doc, nil
}

// DeleteDocument removes a document and its chunks.
func (ing *Ingester) DeleteDocument(kbID, docID string) error {
	if err := ValidateID(kbID); err != nil {
		return err
	}
	if err := ValidateID(docID); err != nil {
		return err
	}
	idx, err := openIndex(ing.mgr.indexPath(kbID))
	if err != nil {
		return err
	}
	defer idx.Close()
	if err := idx.deleteDocument(docID); err != nil {
		return err
	}
	matches, _ := filepath.Glob(filepath.Join(ing.mgr.docsDir(kbID), docID+"_*"))
	for _, p := range matches {
		_ = os.Remove(p)
	}
	return ing.mgr.touch(kbID)
}

// ListDocuments returns documents in a knowledge base.
func (ing *Ingester) ListDocuments(kbID string) ([]Document, error) {
	if err := ValidateID(kbID); err != nil {
		return nil, err
	}
	idx, err := openIndex(ing.mgr.indexPath(kbID))
	if err != nil {
		return nil, err
	}
	defer idx.Close()
	return idx.listDocuments()
}

// Reindex rebuilds the index from on-disk docs for one KB.
func (ing *Ingester) Reindex(ctx context.Context, kbID string) error {
	if err := ValidateID(kbID); err != nil {
		return err
	}
	if _, err := ing.mgr.readMeta(kbID); err != nil {
		return err
	}
	docsPath := ing.mgr.docsDir(kbID)
	entries, err := os.ReadDir(docsPath)
	if err != nil {
		return fmt.Errorf("read docs: %w", err)
	}
	// Reset index.
	_ = os.Remove(ing.mgr.indexPath(kbID))
	idx, err := openIndex(ing.mgr.indexPath(kbID))
	if err != nil {
		return err
	}
	_ = idx.Close()

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		docID, filename, ok := splitDocFilename(name)
		if !ok {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(docsPath, name))
		if err != nil {
			return err
		}
		info, _ := e.Info()
		created := time.Now().UTC()
		if info != nil {
			created = info.ModTime().UTC()
		}
		doc := Document{ID: docID, Filename: filename, Size: int64(len(raw)), CreatedAt: created}
		if err := ing.indexDocument(ctx, kbID, doc, string(raw)); err != nil {
			return fmt.Errorf("reindex %s: %w", name, err)
		}
	}
	return ing.mgr.touch(kbID)
}

func (ing *Ingester) indexDocument(ctx context.Context, kbID string, doc Document, text string) error {
	if ing.emb == nil {
		return fmt.Errorf("embedder is not configured")
	}
	parts := Chunk(text)
	if len(parts) == 0 {
		return fmt.Errorf("no chunks produced")
	}
	vecs, err := ing.emb.Embed(ctx, parts)
	if err != nil {
		return fmt.Errorf("embed: %w", err)
	}
	if len(vecs) != len(parts) {
		return fmt.Errorf("embed returned %d vectors for %d chunks", len(vecs), len(parts))
	}
	rows := make([]chunkRow, len(parts))
	for i := range parts {
		rows[i] = chunkRow{Index: i, Text: parts[i], Embedding: vecs[i]}
	}
	idx, err := openIndex(ing.mgr.indexPath(kbID))
	if err != nil {
		return err
	}
	defer idx.Close()
	return idx.upsertDocument(doc, rows)
}

func newDocID() string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("doc_%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("doc_%x", b)
}

func sanitizeFilename(name string) string {
	base := filepath.Base(name)
	base = strings.ReplaceAll(base, "..", "_")
	base = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			return r
		case r == '.' || r == '-' || r == '_' || r == ' ':
			return r
		default:
			return '_'
		}
	}, base)
	if base == "" || base == "." {
		return "document.txt"
	}
	return base
}

func splitDocFilename(name string) (docID, filename string, ok bool) {
	// doc_<hex>_original.ext
	const prefix = "doc_"
	if !strings.HasPrefix(name, prefix) {
		return "", "", false
	}
	rest := name[len(prefix):]
	i := strings.IndexByte(rest, '_')
	if i <= 0 {
		return "", "", false
	}
	return prefix + rest[:i], rest[i+1:], true
}
