package core

import (
	"context"
	"time"
)

// Provider is the interface that LLM providers must implement.
// This minimal interface is defined here for Client use.
// See providers package for full implementations.
type Provider interface {
	// ID returns the provider identifier (e.g., "openai", "anthropic").
	ID() string

	// Chat sends a non-streaming chat request.
	Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)

	// StreamChat sends a streaming chat request.
	StreamChat(ctx context.Context, req *ChatRequest) (*ChatStream, error)
}

// ChatStream represents a streaming response from a provider.
type ChatStream struct {
	// Ch receives incremental content chunks.
	Ch <-chan ChatChunk

	// Err receives any error that occurs during streaming.
	Err <-chan error

	// Final receives the complete response when streaming ends.
	Final <-chan *ChatResponse
}

// Client is the main entry point for interacting with LLM providers.
// Client is safe for concurrent use.
type Client struct {
	provider  Provider
	telemetry TelemetryHook
	retry     RetryPolicy
}

// ClientOption configures a Client.
type ClientOption func(*Client)

// NewClient creates a new Client with the given provider and options.
func NewClient(p Provider, opts ...ClientOption) *Client {
	c := &Client{
		provider:  p,
		telemetry: NoopTelemetryHook{},
		retry:     DefaultRetryPolicy(),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// WithTelemetry sets the telemetry hook for the client.
func WithTelemetry(h TelemetryHook) ClientOption {
	return func(c *Client) {
		if h != nil {
			c.telemetry = h
		}
	}
}

// WithRetryPolicy sets the retry policy for the client.
func WithRetryPolicy(r RetryPolicy) ClientOption {
	return func(c *Client) {
		if r != nil {
			c.retry = r
		}
	}
}

// Chat returns a ChatBuilder for constructing and executing a chat request.
func (c *Client) Chat(model ModelID) *ChatBuilder {
	return &ChatBuilder{
		client: c,
		req: ChatRequest{
			Model: model,
		},
	}
}

// ChatBuilder provides a fluent API for building chat requests.
// ChatBuilder is NOT thread-safe and should not be shared across goroutines.
type ChatBuilder struct {
	client *Client
	req    ChatRequest
}

// System appends a system message.
func (b *ChatBuilder) System(s string) *ChatBuilder {
	b.req.Messages = append(b.req.Messages, Message{Role: RoleSystem, Content: s})
	return b
}

// User appends a user message.
func (b *ChatBuilder) User(s string) *ChatBuilder {
	b.req.Messages = append(b.req.Messages, Message{Role: RoleUser, Content: s})
	return b
}

// Assistant appends an assistant message.
func (b *ChatBuilder) Assistant(s string) *ChatBuilder {
	b.req.Messages = append(b.req.Messages, Message{Role: RoleAssistant, Content: s})
	return b
}

// Temperature sets the temperature parameter.
func (b *ChatBuilder) Temperature(v float32) *ChatBuilder {
	b.req.Temperature = &v
	return b
}

// MaxTokens sets the maximum tokens parameter.
func (b *ChatBuilder) MaxTokens(n int) *ChatBuilder {
	b.req.MaxTokens = &n
	return b
}

// Tools sets the tools available for the request.
func (b *ChatBuilder) Tools(ts ...Tool) *ChatBuilder {
	b.req.Tools = ts
	return b
}

// validate checks that the request is valid.
func (b *ChatBuilder) validate() error {
	if b.req.Model == "" {
		return ErrModelRequired
	}
	if len(b.req.Messages) == 0 {
		return ErrNoMessages
	}
	return nil
}

// GetResponse executes the chat request and returns the response.
// It applies validation, telemetry, and retry logic.
func (b *ChatBuilder) GetResponse(ctx context.Context) (*ChatResponse, error) {
	if err := b.validate(); err != nil {
		return nil, err
	}

	start := time.Now()
	providerID := b.client.provider.ID()

	// Emit telemetry start
	b.client.telemetry.OnRequestStart(RequestStartEvent{
		Provider: providerID,
		Model:    b.req.Model,
		Start:    start,
	})

	var resp *ChatResponse
	var err error

	// Execute with retry logic
	for attempt := 0; ; attempt++ {
		resp, err = b.client.provider.Chat(ctx, &b.req)
		if err == nil {
			break
		}

		// Check if we should retry
		delay, shouldRetry := b.client.retry.NextDelay(attempt, err)
		if !shouldRetry {
			break
		}

		// Wait before retry, respecting context cancellation
		select {
		case <-ctx.Done():
			err = ctx.Err()
			break
		case <-time.After(delay):
			continue
		}
		break
	}

	// Emit telemetry end
	end := time.Now()
	usage := TokenUsage{}
	if resp != nil {
		usage = resp.Usage
	}
	b.client.telemetry.OnRequestEnd(RequestEndEvent{
		Provider: providerID,
		Model:    b.req.Model,
		Start:    start,
		End:      end,
		Usage:    usage,
		Err:      err,
	})

	return resp, err
}

// Stream executes the chat request and returns a streaming response.
// It applies validation and telemetry.
func (b *ChatBuilder) Stream(ctx context.Context) (*ChatStream, error) {
	if err := b.validate(); err != nil {
		return nil, err
	}

	start := time.Now()
	providerID := b.client.provider.ID()

	// Emit telemetry start
	b.client.telemetry.OnRequestStart(RequestStartEvent{
		Provider: providerID,
		Model:    b.req.Model,
		Start:    start,
	})

	stream, err := b.client.provider.StreamChat(ctx, &b.req)
	if err != nil {
		// Emit telemetry end on immediate error
		b.client.telemetry.OnRequestEnd(RequestEndEvent{
			Provider: providerID,
			Model:    b.req.Model,
			Start:    start,
			End:      time.Now(),
			Err:      err,
		})
		return nil, err
	}

	// Wrap the stream to emit telemetry when it completes
	return wrapStreamWithTelemetry(stream, b.client.telemetry, providerID, b.req.Model, start), nil
}

// wrapStreamWithTelemetry wraps a ChatStream to emit telemetry on completion.
func wrapStreamWithTelemetry(
	stream *ChatStream,
	hook TelemetryHook,
	provider string,
	model ModelID,
	start time.Time,
) *ChatStream {
	finalCh := make(chan *ChatResponse, 1)
	errCh := make(chan error, 1)

	go func() {
		defer close(finalCh)
		defer close(errCh)

		var finalResp *ChatResponse
		var finalErr error

		// Wait for either final response or error
		select {
		case resp, ok := <-stream.Final:
			if ok {
				finalResp = resp
				finalCh <- resp
			}
		case err, ok := <-stream.Err:
			if ok {
				finalErr = err
				errCh <- err
			}
		}

		// Emit telemetry end
		usage := TokenUsage{}
		if finalResp != nil {
			usage = finalResp.Usage
		}
		hook.OnRequestEnd(RequestEndEvent{
			Provider: provider,
			Model:    model,
			Start:    start,
			End:      time.Now(),
			Usage:    usage,
			Err:      finalErr,
		})
	}()

	return &ChatStream{
		Ch:    stream.Ch,
		Err:   errCh,
		Final: finalCh,
	}
}
