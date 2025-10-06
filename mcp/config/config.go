package config

import (
	"fmt"
	"github.com/viant/fluxor"
	"github.com/viant/fluxor/model/types"
	"github.com/viant/x"
	"os"

	"gopkg.in/yaml.v3"

	mcp "github.com/viant/mcp"
)

type Group[T any] struct {
	URL   string `yaml:"url,omitempty" json:"url,omitempty" short:"u" long:"url" description:"url"`
	Items []T    `yaml:"items,omitempty" json:"items,omitempty" short:"i" long:"items" description:"items"`
}

type Config struct {
	Server         *mcp.ServerOptions `yaml:"server,omitempty" json:"server,omitempty"`
	Options        []fluxor.Option
	Extensions     []types.Service
	ExtensionTypes []*x.Type
	Builtins       []string           `yaml:"builtins,omitempty" json:"builtins,omitempty"`
	MCP            *Group[*MCPClient] `yaml:"mcp,omitempty" json:"mcp,omitempty"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %q: %w", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file %q: %w", path, err)
	}
	return &cfg, nil
}

func (c *Config) Validate() error {
	// ServicePackages are optional now.
	return nil
}

// MCPClient augments mcp.ClientOptions with optional description overrides
// for discovery tools (resources and prompts). The map keys should use
// path-style identifiers relative to the discovery namespace, e.g.:
//   - "resources/list"
//   - "resources/read"
//   - "resources/templates/list"
//   - "prompts/list"
//   - "prompts/get"
//
// When set, these override the default method descriptions.
type MCPClient struct {
	*mcp.ClientOptions `yaml:",inline" json:",inline"`
	Descriptions       map[string]string `yaml:"descriptions,omitempty" json:"descriptions,omitempty"`
	// Metadata is an arbitrary key/value map injected into discovery responses
	// under meta["metadata"]. This can be used by MCP hosts to receive
	// any side-channel information.
	Metadata map[string]interface{} `yaml:"metadata,omitempty" json:"metadata,omitempty"`
}
