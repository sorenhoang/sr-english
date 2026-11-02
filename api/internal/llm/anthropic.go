// Package llm wraps the Anthropic API behind the small interface enrich needs.
package llm

import (
	"context"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

type Anthropic struct {
	client    anthropic.Client
	model     string
	maxTokens int64
}

func NewAnthropic(apiKey, model string, maxTokens int64) Anthropic {
	return Anthropic{client: anthropic.NewClient(option.WithAPIKey(apiKey)), model: model, maxTokens: maxTokens}
}

func (a Anthropic) Complete(ctx context.Context, prompt string) (string, error) {
	msg, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(a.model),
		MaxTokens: a.maxTokens,
		Messages:  []anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock(prompt))},
	})
	if err != nil {
		return "", fmt.Errorf("anthropic messages: %w", err)
	}
	var b strings.Builder
	for _, block := range msg.Content {
		if block.Type == "text" {
			b.WriteString(block.Text)
		}
	}
	return b.String(), nil
}
