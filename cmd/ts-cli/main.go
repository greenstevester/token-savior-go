// Command ts-cli provides offline helpers around the token-savior internals.
//
// Subcommands:
//
//	manifest [--json]   Print manifest byte sizes for each profile.
//	                    Used by the M1 exit gate (Go ≤ Python).
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"

	"token-savior-go/internal/tools"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "manifest":
		runManifest(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: ts-cli <subcommand> [args]")
	fmt.Fprintln(os.Stderr, "subcommands: manifest")
}

func runManifest(args []string) {
	fs := flag.NewFlagSet("manifest", flag.ExitOnError)
	jsonOut := fs.Bool("json", false, "JSON output (default: human-readable table)")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	registry := tools.DefaultRegistry()
	profiles := []struct {
		Name string
		P    tools.ProfileSet
	}{
		{"full", tools.ProfileFull},
		{"core", tools.ProfileCore},
		{"nav", tools.ProfileNav},
		{"lean", tools.ProfileLean},
		{"ultra", tools.ProfileUltra},
		{"tiny", tools.ProfileTiny},
		{"tiny_plus", tools.ProfileTinyPlus},
	}
	out := make(map[string]map[string]int)
	for _, p := range profiles {
		visible := tools.VisibleTools(registry, p.P)
		nameBytes := 0
		descBytes := 0
		schemaBytes := 0
		for _, s := range visible {
			nameBytes += len(s.Name)
			descBytes += len(s.Description)
			schemaBytes += len(s.InputSchema)
		}
		out[p.Name] = map[string]int{
			"tool_count":   len(visible),
			"name_bytes":   nameBytes,
			"desc_bytes":   descBytes,
			"schema_bytes": schemaBytes,
			"total_bytes":  nameBytes + descBytes + schemaBytes,
		}
	}

	if *jsonOut {
		if err := json.NewEncoder(os.Stdout).Encode(out); err != nil {
			fmt.Fprintf(os.Stderr, "encode: %v\n", err)
			os.Exit(1)
		}
		return
	}
	names := make([]string, 0, len(out))
	for n := range out {
		names = append(names, n)
	}
	sort.Strings(names)
	fmt.Printf("%-12s %6s %10s %12s %12s %12s\n", "profile", "tools", "name_b", "desc_b", "schema_b", "total_b")
	for _, n := range names {
		row := out[n]
		fmt.Printf("%-12s %6d %10d %12d %12d %12d\n",
			n, row["tool_count"], row["name_bytes"], row["desc_bytes"],
			row["schema_bytes"], row["total_bytes"])
	}
}
