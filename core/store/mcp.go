package store

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/teexue/nexakit/mcp"
)

// LoadGlobalMCP returns all global MCP server configs.
func (db *DB) LoadGlobalMCP() ([]mcp.ServerConfig, error) {
	var rows []MCPServerRow
	if err := db.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]mcp.ServerConfig, 0, len(rows))
	for _, r := range rows {
		var cfg mcp.ServerConfig
		if err := json.Unmarshal([]byte(r.SpecJSON), &cfg); err != nil {
			return nil, fmt.Errorf("parse mcp %q: %w", r.Name, err)
		}
		cfg.Name = r.Name
		out = append(out, cfg)
	}
	return out, nil
}

// UpsertGlobalMCP adds or replaces a global MCP server by name.
func (db *DB) UpsertGlobalMCP(srv mcp.ServerConfig) error {
	if srv.Name == "" {
		return fmt.Errorf("mcp server name is required")
	}
	b, err := json.Marshal(srv)
	if err != nil {
		return fmt.Errorf("marshal mcp: %w", err)
	}
	return db.Save(&MCPServerRow{Name: srv.Name, SpecJSON: string(b)}).Error
}

// DeleteGlobalMCP removes a global MCP server by name.
func (db *DB) DeleteGlobalMCP(name string) error {
	res := db.Where("name = ?", name).Delete(&MCPServerRow{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("global mcp server %q: %w", name, os.ErrNotExist)
	}
	return nil
}
