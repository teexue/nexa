package config

import (
	"fmt"

	"github.com/teexue/nexakit/mcp"
)

// LoadGlobalMCP reads global MCP servers from SQLite.
func LoadGlobalMCP(home string) ([]mcp.ServerConfig, error) {
	_ = home
	db, err := requireDB()
	if err != nil {
		return nil, err
	}
	servers, err := db.LoadGlobalMCP()
	if err != nil {
		return nil, err
	}
	if servers == nil {
		return []mcp.ServerConfig{}, nil
	}
	return servers, nil
}

// SaveGlobalMCP replaces the full list of global MCP servers.
func SaveGlobalMCP(home string, servers []mcp.ServerConfig) error {
	_ = home
	db, err := requireDB()
	if err != nil {
		return err
	}
	existing, err := db.LoadGlobalMCP()
	if err != nil {
		return err
	}
	for _, s := range existing {
		_ = db.DeleteGlobalMCP(s.Name)
	}
	for _, s := range servers {
		if err := db.UpsertGlobalMCP(s); err != nil {
			return err
		}
	}
	return nil
}

// UpsertGlobalMCP adds or replaces a global MCP server by name.
func UpsertGlobalMCP(home string, srv mcp.ServerConfig) error {
	_ = home
	if srv.Name == "" {
		return fmt.Errorf("mcp server name is required")
	}
	db, err := requireDB()
	if err != nil {
		return err
	}
	return db.UpsertGlobalMCP(srv)
}

// DeleteGlobalMCP removes a global MCP server by name.
func DeleteGlobalMCP(home, name string) error {
	_ = home
	db, err := requireDB()
	if err != nil {
		return err
	}
	return db.DeleteGlobalMCP(name)
}
