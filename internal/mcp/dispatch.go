package mcp

import (
	"encoding/json"
	"fmt"
)

// Dispatcher routes tool names to handlers.
type Dispatcher struct {
	ctx      *ToolContext
	handlers map[string]Handler
}

// NewDispatcher returns an empty dispatcher bound to ctx.
func NewDispatcher(ctx *ToolContext) *Dispatcher {
	return &Dispatcher{ctx: ctx, handlers: make(map[string]Handler)}
}

// Register installs handler under name. Re-registering a name overwrites
// the previous handler (callers normally register only at startup).
func (d *Dispatcher) Register(name string, handler Handler) {
	d.handlers[name] = handler
}

// Dispatch invokes the registered handler for name. Returns (nil, error)
// when no handler is registered. The Stats counter for the tool is
// incremented on every call regardless of success, with chars measured
// from the JSON-encoded result.
func (d *Dispatcher) Dispatch(name string, args json.RawMessage) (any, error) {
	handler, ok := d.handlers[name]
	if !ok {
		return nil, fmt.Errorf("unknown tool: %s", name)
	}

	result, err := handler(d.ctx, args)

	// Record the call regardless of error; char count is best-effort.
	chars := int64(0)
	if result != nil {
		if b, jerr := json.Marshal(result); jerr == nil {
			chars = int64(len(b))
		}
	}
	d.ctx.Stats.RecordCall(name, chars)
	return result, err
}
