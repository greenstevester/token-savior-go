// Command ts-compat runs Python v3 and Go v4 token-savior side-by-side
// on a fixture project and diffs structured tool outputs.
//
// Usage:
//
//	ts-compat -fixture testdata/fixtures/go-small -python token-savior
//
// Requires:
//   - Python `token-savior` on PATH (or via -python flag). When absent the
//     binary exits 0 with a "skipped: …" message so the harness doesn't
//     break local builds before Python is installed.
//   - This binary built (`make build-ts-compat`).
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"token-savior-go/internal/compat"
)

func main() {
	fixture := flag.String("fixture", "", "absolute path to fixture project")
	pythonBin := flag.String("python", "token-savior", "Python token-savior binary on PATH")
	goBin := flag.String("go", "./bin/token-savior", "Go token-savior binary")
	flag.Parse()

	if *fixture == "" {
		fmt.Fprintln(os.Stderr, "ts-compat: -fixture is required")
		os.Exit(2)
	}

	// Skip cleanly if Python token-savior isn't available — keeps the harness
	// runnable on machines without the v3 install (CI without Python yet, etc.).
	if _, err := exec.LookPath(*pythonBin); err != nil {
		fmt.Printf("skipped: %q not on PATH (install token-savior-recall and re-run)\n", *pythonBin)
		return
	}

	tools := []toolCase{
		{Name: "find_symbol", Args: map[string]any{"name": "Greeter.Hello", "compress": false}},
		{Name: "get_functions", Args: map[string]any{"compress": false}},
		{Name: "get_classes", Args: map[string]any{"compress": false}},
		{Name: "get_imports", Args: map[string]any{"compress": false}},
		{Name: "search_codebase", Args: map[string]any{"pattern": "Greeter", "regex": false}},
		{Name: "list_workspace_roots", Args: map[string]any{}},
	}

	failures := 0
	for _, tc := range tools {
		pyOut, err := callTool(*pythonBin, *fixture, tc.Name, tc.Args)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[python %s] %v\n", tc.Name, err)
			failures++
			continue
		}
		goOut, err := callTool(*goBin, *fixture, tc.Name, tc.Args)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[go %s] %v\n", tc.Name, err)
			failures++
			continue
		}
		diffs, err := compat.DiffJSON(pyOut, goOut)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[diff %s] %v\n", tc.Name, err)
			failures++
			continue
		}
		if len(diffs) == 0 {
			fmt.Printf("OK   %s\n", tc.Name)
			continue
		}
		fmt.Printf("DIFF %s\n", tc.Name)
		for _, d := range diffs {
			fmt.Printf("  - %s\n", d)
		}
		failures++
	}

	if failures > 0 {
		fmt.Fprintf(os.Stderr, "\n%d tool(s) failed parity\n", failures)
		os.Exit(1)
	}
	fmt.Println("\nAll tools matched")
}

type toolCase struct {
	Name string
	Args map[string]any
}

// callTool runs the binary as an MCP stdio subprocess, calls tool with args,
// and returns the JSON-text content of the response. Times out after 30s.
//
// MCP framing notes: raw JSON-RPC over stdio. Each request is a single line
// of JSON; response is parsed from the first JSON line on stdout that has
// "id" == 2. Minimalist — full MCP framing isn't needed for one-shot calls.
func callTool(bin, root, tool string, args map[string]any) (json.RawMessage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, bin) //nolint:gosec // bin is user-supplied via flag; harness is a dev tool, not a service.
	cmd.Env = append(os.Environ(), "WORKSPACE_ROOTS="+root, "TOKEN_SAVIOR_PROFILE=full", "TS_NO_HINTS=1")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	// Minimal MCP handshake + tools/call. JSON-RPC frames may need adjusting
	// per SDK version — this is the highest-risk part of the harness.
	requests := []string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"clientInfo":{"name":"ts-compat","version":"0.1"},"protocolVersion":"2024-11-05","capabilities":{}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized","params":{}}`,
		fmt.Sprintf(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":%q,"arguments":%s}}`, tool, mustJSON(args)),
	}
	for _, r := range requests {
		if _, werr := io.WriteString(stdin, r+"\n"); werr != nil {
			return nil, werr
		}
	}
	_ = stdin.Close()

	decoder := json.NewDecoder(stdout)
	for {
		var frame map[string]any
		if derr := decoder.Decode(&frame); derr != nil {
			if errors.Is(derr, io.EOF) {
				return nil, fmt.Errorf("stdout closed before response (process exited?)")
			}
			return nil, fmt.Errorf("decode response: %w", derr)
		}
		idVal, ok := frame["id"].(float64)
		if !ok || idVal != 2 {
			continue
		}
		result, ok := frame["result"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("no result in response: %v", frame)
		}
		content, ok := result["content"].([]any)
		if !ok || len(content) == 0 {
			return nil, fmt.Errorf("no content: %v", result)
		}
		item, ok := content[0].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("bad content shape: %v", content)
		}
		text, textOK := item["text"].(string)
		if !textOK {
			text = ""
		}
		return json.RawMessage(strings.TrimSpace(text)), nil
	}
}

func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		// Only reachable with non-serialisable types; all callers pass map[string]any.
		return "null"
	}
	return string(b)
}
