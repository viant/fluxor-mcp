package config

import (
	"fmt"
	"github.com/viant/fluxor"
	"github.com/viant/fluxor/model/types"
	"github.com/viant/x"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/viant/mcp"
)

type Group[T any] struct {
	URL   string `yaml:"url" json:"url" short:"u" long:"url" description:"url"`
	Items []T    `yaml:"items" json:"items" short:"i" long:"items" description:"items"`
}

type Config struct {
	Server         *mcp.ServerOptions `yaml:"server" json:"server"`
	Options        []fluxor.Option
	Extensions     []types.Service
	ExtensionTypes []*x.Type
    Builtins       []string `yaml:"builtins" json:"builtins"`
	MCP            *Group[*mcp.ClientOptions] `yaml:"mcp" json:"mcp"`
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
