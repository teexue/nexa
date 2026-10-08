package agent

import (
	"log/slog"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
)

// ChangeType identifies what kind of file change occurred.
type ChangeType string

const (
	// ChangeCreated is emitted when a new agent YAML appears on disk so the
	// catalog can load it without a process restart.
	ChangeCreated ChangeType = "created"
	// ChangeUpdated is emitted when an existing agent YAML is rewritten so the
	// running catalog can hot-reload the definition.
	ChangeUpdated ChangeType = "updated"
	// ChangeDeleted is emitted when an agent YAML is removed or renamed away,
	// so the catalog drops that agent.
	ChangeDeleted ChangeType = "deleted"
)

// AgentChange describes a change to an agent YAML file.
type AgentChange struct {
	Name string     // agent name (filename without .yaml)
	Type ChangeType // created, updated, deleted
	Path string     // full file path
}

// ChangeHandler is called when an agent file changes.
type Handler func(change AgentChange)

// Watcher monitors the agents directory for file changes.
type Watcher struct {
	dir     string
	logger  *slog.Logger
	handler Handler

	mu      sync.Mutex
	watcher *fsnotify.Watcher
	done    chan struct{}
}

// NewWatcher creates a file watcher for the agents directory.
func NewWatcher(dir string, logger *slog.Logger, handler Handler) *Watcher {
	if logger == nil {
		logger = slog.Default()
	}
	return &Watcher{
		dir:     dir,
		logger:  logger,
		handler: handler,
		done:    make(chan struct{}),
	}
}

// Start begins watching the agents directory for changes.
func (w *Watcher) Start() error {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	if err := fw.Add(w.dir); err != nil {
		fw.Close()
		return err
	}

	w.mu.Lock()
	w.watcher = fw
	w.mu.Unlock()

	go w.loop()
	w.logger.Info("log.agent_watcher.started", "dir", w.dir)
	return nil
}

// Stop stops the watcher.
func (w *Watcher) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()

	select {
	case <-w.done:
		return // already stopped
	default:
		close(w.done)
	}

	if w.watcher != nil {
		w.watcher.Close()
		w.watcher = nil
	}
	w.logger.Info("log.agent_watcher.stopped")
}

func (w *Watcher) loop() {
	// Stop nils the field as soon as it runs. Keep a local copy so a stop
	// that wins the race does not dereference nil on the next select.
	w.mu.Lock()
	fw := w.watcher
	w.mu.Unlock()
	if fw == nil {
		return
	}
	for {
		select {
		case <-w.done:
			return
		case event, ok := <-fw.Events:
			if !ok {
				return
			}
			w.handleEvent(event)
		case err, ok := <-fw.Errors:
			if !ok {
				return
			}
			w.logger.Error("log.agent_watcher.error", "error", err)
		}
	}
}

func (w *Watcher) handleEvent(event fsnotify.Event) {
	// Only care about .yaml files.
	name := filepath.Base(event.Name)
	if !strings.HasSuffix(name, ".yaml") {
		return
	}

	agentName := strings.TrimSuffix(name, ".yaml")

	var changeType ChangeType
	switch {
	case event.Op&(fsnotify.Create|fsnotify.Write) != 0:
		// Distinguish create vs update by checking if the file was just created.
		if event.Op&fsnotify.Create != 0 {
			changeType = ChangeCreated
		} else {
			changeType = ChangeUpdated
		}
	case event.Op&fsnotify.Remove != 0:
		changeType = ChangeDeleted
	case event.Op&fsnotify.Rename != 0:
		changeType = ChangeDeleted
	default:
		return
	}

	change := AgentChange{
		Name: agentName,
		Type: changeType,
		Path: event.Name,
	}

	w.logger.Debug("log.agent_watcher.file_changed", "name", agentName, "type", changeType, "op", event.Op.String())

	if w.handler != nil {
		w.handler(change)
	}
}
