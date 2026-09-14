package store

import (
	"github.com/teexue/nexakit/embedding"
	"github.com/teexue/nexakit/mcp"
)

type fileEmbedding struct {
	Vendor         string `yaml:"vendor,omitempty"`
	Backend        string `yaml:"backend"`
	BaseURL        string `yaml:"base_url"`
	APIKeyEnv      string `yaml:"api_key_env,omitempty"`
	Model          string `yaml:"model"`
	Dimensions     int    `yaml:"dimensions,omitempty"`
	LegacyProvider string `yaml:"provider,omitempty"`
}

func (e fileEmbedding) toConfig() embedding.Config {
	return embedding.Config{
		Vendor: e.Vendor, Backend: e.Backend, BaseURL: e.BaseURL,
		APIKeyEnv: e.APIKeyEnv, Model: e.Model, Dimensions: e.Dimensions,
		LegacyProvider: e.LegacyProvider,
	}
}

type fileMCPServer struct {
	Name    string            `yaml:"name"`
	Type    string            `yaml:"type"`
	Command string            `yaml:"command,omitempty"`
	Args    []string          `yaml:"args,omitempty"`
	Env     map[string]string `yaml:"env,omitempty"`
	URL     string            `yaml:"url,omitempty"`
}

func (s fileMCPServer) toConfig() mcp.ServerConfig {
	return mcp.ServerConfig{
		Name: s.Name, Type: s.Type, Command: s.Command,
		Args: s.Args, Env: s.Env, URL: s.URL,
	}
}
