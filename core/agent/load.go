package agent

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Load reads and validates an agent YAML file.
// Legacy files without an id field use the filename stem as the id.
func Load(path string) (*Agent, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read agent %q: %w", path, err)
	}
	a, err := LoadFromBytes(data)
	if err != nil {
		return nil, err
	}
	stem := strings.TrimSuffix(filepath.Base(path), ".yaml")
	if a.ID == "" {
		a.ID = stem
	}
	return a, nil
}

// LoadFromBytes parses and validates an agent from raw YAML bytes.
// ID may be empty (assigned on save / from filename on Load).
func LoadFromBytes(data []byte) (*Agent, error) {
	return parseFile(data)
}

// LoadByID loads agents/{id}.yaml from dir.
func LoadByID(dir, id string) (*Agent, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("agent id is required")
	}
	path := filepath.Join(dir, id+".yaml")
	return Load(path)
}

// LoadByName resolves an agent by id or display name (legacy compatibility).
func LoadByName(dir, ref string) (*Agent, error) {
	return Resolve(dir, ref)
}

// Resolve loads an agent by stable id (filename) or display name.
// Id match takes priority; name match scans all agents.
func Resolve(dir, ref string) (*Agent, error) {
	ref = strings.TrimSuffix(strings.TrimSpace(ref), ".yaml")
	if ref == "" {
		return nil, fmt.Errorf("agent ref is required")
	}
	path := filepath.Join(dir, ref+".yaml")
	if _, err := os.Stat(path); err == nil {
		return Load(path)
	}
	result, err := LoadAll(dir)
	if err != nil {
		return nil, err
	}
	var byName *Agent
	for _, a := range result.Agents {
		if a.ID == ref {
			return a, nil
		}
		if a.Name == ref && byName == nil {
			byName = a
		}
	}
	if byName != nil {
		return byName, nil
	}
	return nil, fmt.Errorf("agent %q: %w", ref, os.ErrNotExist)
}

// ResolveDefault loads preferred when it exists; otherwise the first agent in
// dir (ids sorted lexicographically). Empty preferred skips straight to the
// first agent. Non-existence of preferred falls back; other load errors do not.
func ResolveDefault(dir, preferred string) (*Agent, error) {
	preferred = strings.TrimSuffix(strings.TrimSpace(preferred), ".yaml")
	if preferred != "" {
		a, err := Resolve(dir, preferred)
		if err == nil {
			return a, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}
	ids, err := ListAvailable(dir)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("no agents configured in %q", dir)
	}
	return LoadByID(dir, ids[0])
}

// ListAvailable returns the ids of all agent YAML files in dir.
func ListAvailable(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read agents dir %q: %w", dir, err)
	}
	var ids []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(e.Name(), ".yaml") {
			ids = append(ids, strings.TrimSuffix(e.Name(), ".yaml"))
		}
	}
	sort.Strings(ids)
	return ids, nil
}

// LoadAndValidate loads an agent and checks referenced tools exist.
func LoadAndValidate(path string, toolNames []string) (*Agent, error) {
	a, err := Load(path)
	if err != nil {
		return nil, err
	}
	if err := validateToolRefs(a, toolNames); err != nil {
		return nil, err
	}
	return a, nil
}

// LoadByNameAndValidate loads an agent by id or name and validates tools.
func LoadByNameAndValidate(dir, ref string, toolNames []string) (*Agent, error) {
	a, err := Resolve(dir, ref)
	if err != nil {
		return nil, err
	}
	if err := validateToolRefs(a, toolNames); err != nil {
		return nil, err
	}
	return a, nil
}

func validateToolRefs(a *Agent, toolNames []string) error {
	nameSet := make(map[string]bool, len(toolNames))
	for _, n := range toolNames {
		nameSet[n] = true
	}
	var missing []string
	for _, t := range a.Tools {
		if !nameSet[t] {
			missing = append(missing, t)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("agent %q references unregistered tools: %v", a.Name, missing)
	}
	return nil
}

// AgentLoadError records a single agent YAML file that failed to load.
type AgentLoadError struct {
	ID   string
	Name string
	Path string
	Err  error
}

// Error implements the error interface.
func (e AgentLoadError) Error() string {
	label := e.ID
	if e.Name != "" {
		label = e.Name
	}
	return fmt.Sprintf("agent %q (%s): %v", label, e.Path, e.Err)
}

// LoadAllResult is the outcome of bulk-loading all agents in a directory.
type LoadAllResult struct {
	Agents []*Agent
	Errors []AgentLoadError
}

// LoadAll loads every .yaml file in dir. It never fails fast.
func LoadAll(dir string) (*LoadAllResult, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read agents dir %q: %w", dir, err)
	}
	var result LoadAllResult
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".yaml")
		path := filepath.Join(dir, e.Name())
		a, err := Load(path)
		if err != nil {
			result.Errors = append(result.Errors, AgentLoadError{ID: id, Path: path, Err: err})
			continue
		}
		result.Agents = append(result.Agents, a)
	}
	sort.Slice(result.Agents, func(i, j int) bool {
		return result.Agents[i].Name < result.Agents[j].Name
	})
	return &result, nil
}

// LoadAllAndValidate loads all agents and drops those with unknown tools.
func LoadAllAndValidate(dir string, toolNames []string) (*LoadAllResult, error) {
	result, err := LoadAll(dir)
	if err != nil {
		return nil, err
	}
	nameSet := make(map[string]bool, len(toolNames))
	for _, n := range toolNames {
		nameSet[n] = true
	}
	var validated []*Agent
	for _, a := range result.Agents {
		var missing []string
		for _, t := range a.Tools {
			if !nameSet[t] {
				missing = append(missing, t)
			}
		}
		if len(missing) > 0 {
			result.Errors = append(result.Errors, AgentLoadError{
				ID: a.ID, Name: a.Name, Path: filepath.Join(dir, a.ID+".yaml"),
				Err: fmt.Errorf("references unregistered tools: %v", missing),
			})
			continue
		}
		validated = append(validated, a)
	}
	result.Agents = validated
	return result, nil
}
