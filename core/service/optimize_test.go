package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/teexue/common-agent/core/agent"
	"github.com/teexue/nexakit/provider"
)

// staticProvider replies with a fixed text on every Stream call.
type staticProvider struct{ text string }

func (s staticProvider) Stream(context.Context, provider.Request) (<-chan provider.Chunk, error) {
	ch := make(chan provider.Chunk, 2)
	ch <- provider.Chunk{TextDelta: s.text}
	ch <- provider.Chunk{Done: true}
	close(ch)
	return ch, nil
}

// doneOnlyProvider returns a stream that completes without any text,
// simulating an empty LLM response.
type doneOnlyProvider struct{}

func (doneOnlyProvider) Stream(context.Context, provider.Request) (<-chan provider.Chunk, error) {
	ch := make(chan provider.Chunk, 1)
	ch <- provider.Chunk{Done: true}
	close(ch)
	return ch, nil
}

func TestOptimizeUserPrompt_Enabled(t *testing.T) {
	a := &agent.Agent{Name: "t", Model: "m", Optimize: &agent.OptimizeConfig{UserPrompt: true}}
	got := OptimizeUserPrompt(context.Background(), a, staticProvider{"优化后的问题"}, "原始问题", nil)
	assert.Equal(t, "优化后的问题", got)
}

func TestOptimizeUserPrompt_Disabled(t *testing.T) {
	a := &agent.Agent{Name: "t", Model: "m"}
	got := OptimizeUserPrompt(context.Background(), a, staticProvider{"优化后的问题"}, "原始问题", nil)
	assert.Equal(t, "原始问题", got)
}

func TestOptimizeUserPrompt_EmptyResponseFallsBack(t *testing.T) {
	a := &agent.Agent{Name: "t", Model: "m", Optimize: &agent.OptimizeConfig{UserPrompt: true}}
	got := OptimizeUserPrompt(context.Background(), a, doneOnlyProvider{}, "原始问题", nil)
	assert.Equal(t, "原始问题", got)
}

func TestOptimizeUserPrompt_SkipsMockProvider(t *testing.T) {
	// MockProvider's scripted responses must be reserved for the actual run.
	a := &agent.Agent{Name: "t", Model: "m", Optimize: &agent.OptimizeConfig{UserPrompt: true}}
	mock := &provider.MockProvider{Calls: [][]provider.MockStep{{{Text: "run reply"}}}}
	got := OptimizeUserPrompt(context.Background(), a, mock, "原始问题", nil)
	assert.Equal(t, "原始问题", got)
}
