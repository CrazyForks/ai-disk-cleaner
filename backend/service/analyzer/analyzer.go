package analyzer

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"ai-disk-cleanner/backend/data/models/cleaningrecord"
	"ai-disk-cleanner/backend/i18n"
	modelscanner "ai-disk-cleanner/backend/model/scanner"

	"github.com/openai/openai-go/v3"
)

// Analyzer implements the LLM analysis service with an OpenAI-compatible API.
type Analyzer struct {
	settings settingStore
}

func NewAnalyzer(settings settingStore) *Analyzer {
	return &Analyzer{settings: settings}
}

func (analyzer *Analyzer) Analyze(
	ctx context.Context,
	tree *modelscanner.FileTree,
	language string,
	onDelta func(string),
) (*cleaningrecord.AnalysisResult, error) {
	if tree == nil {
		return nil, errors.New("analyze disk: file tree is nil")
	}
	if onDelta == nil {
		onDelta = func(string) {}
	}
	config, err := analyzer.loadLLMConfig(ctx)
	if err != nil {
		return nil, err
	}
	client := newOpenAIClient(config)

	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(systemPrompt),
		openai.UserMessage(i18n.AnalyzerUserPrompt(language)),
	}
	manager := newManager()
	diskContext := newDiskCleanerContext(tree)
	llmTools := buildTools(manager)

	var output strings.Builder
	var tokenUsage int64
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		stream := client.Chat.Completions.NewStreaming(
			ctx,
			openai.ChatCompletionNewParams{
				Messages:  messages,
				Model:     config.model,
				Tools:     llmTools,
				MaxTokens: openai.Int(config.maxTokens),
				StreamOptions: openai.ChatCompletionStreamOptionsParam{
					IncludeUsage: openai.Bool(true),
				},
			},
		)
		accumulator := openai.ChatCompletionAccumulator{}
		for stream.Next() {
			chunk := stream.Current()
			if !accumulator.AddChunk(chunk) {
				_ = stream.Close()
				return nil, errors.New("analyze disk: could not accumulate LLM stream")
			}
			if len(chunk.Choices) > 0 {
				delta := chunk.Choices[0].Delta.Content
				if delta != "" {
					output.WriteString(delta)
					onDelta(delta)
				}
			}
		}
		streamErr := stream.Err()
		_ = stream.Close()
		if streamErr != nil {
			return nil, fmt.Errorf("analyze disk stream: %w", streamErr)
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if len(accumulator.Choices) == 0 {
			return nil, errors.New("analyze disk: LLM returned no choices")
		}

		tokenUsage += accumulator.Usage.CompletionTokens
		message := accumulator.Choices[0].Message
		if len(message.ToolCalls) == 0 {
			break
		}
		messages = append(messages, message.ToParam())
		for _, item := range message.ToolCalls {
			function := item.Function
			result, err := manager.Invoke(
				function.Name,
				function.Arguments,
				diskContext,
			)
			if err != nil {
				result = err.Error()
			}
			messages = append(messages, openai.ToolMessage(result, item.ID))
		}
	}

	return &cleaningrecord.AnalysisResult{
		TrashFiles: diskContext.TrashFiles,
		TopUsages:  diskContext.TopUsages,
		LLMOutput:  output.String(),
		TokenUsage: tokenUsage,
	}, nil
}
