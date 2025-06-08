package cmd

import (
	"context"
	"encoding/json"
	"os"
	"sync"

	"github.com/viant/fluxor-mcp/mcp"
	mcpconfig "github.com/viant/fluxor-mcp/mcp/config"
)

var (
	cfgPath string

	svcOnce sync.Once
	svcInst *mcp.Service
	svcErr  error
)

// setConfigPath remembers the CLI-level -f/--config parameter so that the
// service singleton can be created lazily by whichever sub-command is executed
// first.
func setConfigPath(p string) { cfgPath = p }

// serviceSingleton initialises an mcp.Service only once and reuses the instance
// across sub-commands within the same CLI invocation.
func serviceSingleton() (*mcp.Service, error) {
	svcOnce.Do(func() {
		var cfg *mcpconfig.Config
		if cfgPath != "" {
			var err error
			cfg, err = mcpconfig.Load(cfgPath)
			if err != nil {
				svcErr = err
				return
			}
			// Pretty-print location if the user asked for it via env for debug.
			if debug := os.Getenv("MCPCLI_DEBUG_CONFIG"); debug == "1" {
				_ = json.NewEncoder(os.Stderr).Encode(cfg)
			}
		}

		svcInst, svcErr = mcp.New(context.Background(), mcp.WithConfig(cfg))
		if svcErr == nil {
			svcErr = svcInst.Start(context.Background())
		}
	})
	return svcInst, svcErr
}
