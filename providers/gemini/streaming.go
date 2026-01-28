package gemini

import (
	"context"

	"github.com/erikhoward/iris/core"
)

// doStreamChat performs a streaming chat request.
// TODO: Full implementation in Task 8.
func (p *Gemini) doStreamChat(ctx context.Context, req *core.ChatRequest) (*core.ChatStream, error) {
	return nil, errNotImplemented
}
