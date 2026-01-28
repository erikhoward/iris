package gemini

import (
	"context"
	"errors"

	"github.com/erikhoward/iris/core"
)

// errNotImplemented is returned when a method is not yet implemented.
var errNotImplemented = errors.New("not implemented")

// doChat performs a non-streaming chat request.
// TODO: Full implementation in Task 7.
func (p *Gemini) doChat(ctx context.Context, req *core.ChatRequest) (*core.ChatResponse, error) {
	return nil, errNotImplemented
}
