package tui

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	maxIndexedFiles = 2000
	maxFileRead     = 64 * 1024
)

var skipDirNames = map[string]struct{}{
	".git": {}, "node_modules": {}, "vendor": {}, "dist": {}, "build": {},
	".cursor": {}, ".nexa": {}, "__pycache__": {}, ".idea": {},
	".vscode": {}, "target": {}, ".next": {}, "coverage": {},
}

// indexWorkdir walks root and returns relative file paths for the @ picker.
func indexWorkdir(root string) []string {
	if root == "" {
		return nil
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return nil
	}
	out := make([]string, 0, 256)
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		name := d.Name()
		if d.IsDir() {
			if _, skip := skipDirNames[name]; skip {
				return filepath.SkipDir
			}
			if strings.HasPrefix(name, ".") && path != root {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(name, ".") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		out = append(out, filepath.ToSlash(rel))
		if len(out) >= maxIndexedFiles {
			return filepath.SkipAll
		}
		return nil
	})
	return out
}

func filterFiles(files []string, query string) []string {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		if len(files) > 80 {
			return files[:80]
		}
		return files
	}
	out := make([]string, 0, 64)
	for _, f := range files {
		if strings.Contains(strings.ToLower(f), q) {
			out = append(out, f)
			if len(out) >= 80 {
				break
			}
		}
	}
	return out
}

// readWorkdirFile reads a relative path under root, truncated for prompts.
func readWorkdirFile(root, rel string) (string, error) {
	rel = filepath.Clean(filepath.FromSlash(rel))
	if strings.HasPrefix(rel, "..") {
		return "", os.ErrPermission
	}
	full := filepath.Join(root, rel)
	data, err := os.ReadFile(full)
	if err != nil {
		return "", err
	}
	if len(data) > maxFileRead {
		data = append(data[:maxFileRead], []byte("\n...[truncated]...")...)
	}
	return string(data), nil
}

// expandFileMentions replaces @path tokens by appending file bodies.
func expandFileMentions(root, prompt string) string {
	if root == "" || !strings.Contains(prompt, "@") {
		return prompt
	}
	var b strings.Builder
	b.WriteString(prompt)
	seen := map[string]struct{}{}
	for _, tok := range strings.Fields(prompt) {
		if !strings.HasPrefix(tok, "@") || len(tok) < 2 {
			continue
		}
		rel := strings.TrimPrefix(tok, "@")
		rel = strings.Trim(rel, "\"'`")
		if _, ok := seen[rel]; ok {
			continue
		}
		seen[rel] = struct{}{}
		body, err := readWorkdirFile(root, rel)
		if err != nil {
			continue
		}
		b.WriteString("\n\n--- file: ")
		b.WriteString(rel)
		b.WriteString(" ---\n")
		b.WriteString(body)
		b.WriteString("\n---\n")
	}
	return b.String()
}

// atTrigger finds an in-progress @query at the end of value.
func atTrigger(value string) (active bool, query, prefix string) {
	i := strings.LastIndex(value, "@")
	if i < 0 {
		return false, "", ""
	}
	if i > 0 {
		prev := value[i-1]
		if prev != ' ' && prev != '\n' && prev != '\t' {
			return false, "", ""
		}
	}
	rest := value[i+1:]
	if strings.ContainsAny(rest, " \n\t") {
		return false, "", ""
	}
	return true, rest, value[:i]
}
