package analyzer

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"ai-disk-cleanner/backend/data/models/setting"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

type settingStore interface {
	ListSettings(context.Context) ([]setting.Setting, error)
}

type llmConfig struct {
	secret    string
	baseURL   string
	model     string
	maxTokens int64
}

func (analyzer *Service) loadLLMConfig(ctx context.Context) (llmConfig, error) {
	if analyzer.settings == nil {
		return llmConfig{}, errors.New("load LLM configuration: setting store is nil")
	}
	settings, err := analyzer.settings.ListSettings(ctx)
	if err != nil {
		return llmConfig{}, fmt.Errorf("load LLM configuration: %w", err)
	}
	values := make(map[string]string, len(settings))
	for _, setting := range settings {
		values[setting.Key] = setting.Value
	}

	config := llmConfig{
		secret:  strings.TrimSpace(values["llm.secret"]),
		baseURL: strings.TrimSpace(values["llm.url"]),
		model:   strings.TrimSpace(values["llm.model"]),
	}
	if config.secret == "" {
		return llmConfig{}, errors.New("load LLM configuration: llm.secret is empty")
	}
	if config.baseURL == "" {
		return llmConfig{}, errors.New("load LLM configuration: llm.url is empty")
	}
	if config.model == "" {
		return llmConfig{}, errors.New("load LLM configuration: llm.model is empty")
	}
	config.maxTokens, err = strconv.ParseInt(strings.TrimSpace(values["llm.max-token"]), 10, 64)
	if err != nil || config.maxTokens <= 0 {
		return llmConfig{}, errors.New("load LLM configuration: llm.max-token must be a positive integer")
	}
	return config, nil
}

func newOpenAIClient(config llmConfig) openai.Client {
	return openai.NewClient(
		option.WithAPIKey(config.secret),
		option.WithBaseURL(config.baseURL),
	)
}
