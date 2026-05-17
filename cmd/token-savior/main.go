// Command token-savior runs the MCP stdio server.
package main

import (
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/server"

	_ "token-savior-go/internal/annotator/golang" // register Go annotator

	"token-savior-go/internal/mcp"
	"token-savior-go/internal/slot"
	"token-savior-go/internal/stats"
	"token-savior-go/internal/tools"
)

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "[token-savior] fatal: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	roots := slot.ParseWorkspaceRoots(os.Getenv("WORKSPACE_ROOTS"))
	if legacy := os.Getenv("PROJECT_ROOT"); legacy != "" && len(roots) == 0 {
		roots = []string{legacy}
	}
	if len(roots) == 0 {
		return fmt.Errorf("set WORKSPACE_ROOTS or PROJECT_ROOT")
	}

	mgr := slot.NewManager()
	for _, r := range roots {
		if err := mgr.RegisterRoot(r); err != nil {
			fmt.Fprintf(os.Stderr, "[token-savior] register %s: %v\n", r, err)
		}
	}

	tctx := &mcp.ToolContext{
		SlotManager: mgr,
		Stats:       stats.NewCounters(),
	}
	dispatcher := mcp.NewDispatcher(tctx)
	mcp.RegisterHandlers(dispatcher)

	registry := tools.DefaultRegistry()
	profile := tools.ParseProfile(os.Getenv("TOKEN_SAVIOR_PROFILE"))

	srv := server.NewMCPServer("token-savior", Version)
	if err := mcp.Serve(srv, dispatcher, registry, profile); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "[token-savior] profile=%s version=%s commit=%s roots=%d\n",
		os.Getenv("TOKEN_SAVIOR_PROFILE"), Version, Commit, len(roots))

	return server.ServeStdio(srv)
}
