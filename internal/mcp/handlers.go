package mcp

// RegisterHandlers wires the M1 tool handlers onto d. T20 fills the body —
// for T19 we ship a stub so cmd/token-savior compiles and runs (it just
// advertises tools/list with no working handlers; every call routes to
// "unknown tool: …").
func RegisterHandlers(d *Dispatcher) {
	_ = d
}
