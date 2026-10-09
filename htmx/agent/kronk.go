//go:build kronk

package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/ardanlabs/kronk/sdk/kronk"
	"github.com/ardanlabs/kronk/sdk/kronk/model"
	"github.com/ardanlabs/kronk/sdk/tools/libs"
	"github.com/ardanlabs/kronk/sdk/tools/models"
)

type kronkGenerator struct {
	client    *kronk.Kronk
	maxTokens int
	once      sync.Once
	closeErr  error
}

var kronkInitialization struct {
	sync.Mutex
	libraryPath string
}

func newKronkGenerator(ctx context.Context, source, cacheDir string, maxTokens int) (Generator, error) {
	libraryManager, err := libs.New(libs.WithBasePath(cacheDir), libs.WithValidation(true))
	if err != nil {
		return nil, fmt.Errorf("agent: configure Kronk libraries: %w", err)
	}
	if _, err := libraryManager.Download(ctx, kronk.FmtLogger); err != nil {
		return nil, fmt.Errorf("agent: install Kronk libraries: %w", err)
	}
	if err := initKronk(libraryManager.LibsPath()); err != nil {
		return nil, err
	}

	modelManager, err := models.NewWithPaths(cacheDir)
	if err != nil {
		return nil, fmt.Errorf("agent: configure Kronk models: %w", err)
	}
	modelPath, err := modelManager.Download(ctx, kronk.FmtLogger, source)
	if err != nil {
		return nil, fmt.Errorf("agent: install Kronk model %q: %w", source, err)
	}
	client, err := kronk.NewWithContext(ctx,
		model.WithModelFiles(modelPath.ModelFiles),
		model.WithAutoTune(true),
	)
	if err != nil {
		return nil, fmt.Errorf("agent: load Kronk model %q: %w", source, err)
	}
	return &kronkGenerator{client: client, maxTokens: maxTokens}, nil
}

func initKronk(libraryPath string) error {
	kronkInitialization.Lock()
	defer kronkInitialization.Unlock()
	if kronk.Initialized() {
		if kronkInitialization.libraryPath == "" {
			return errors.New("agent: Kronk was initialized outside the agent package; initialize it with the same library path before creating an agent")
		}
		if kronkInitialization.libraryPath != libraryPath {
			return fmt.Errorf("agent: Kronk already uses native libraries at %q, cannot switch to %q", kronkInitialization.libraryPath, libraryPath)
		}
		return nil
	}
	if err := kronk.Init(kronk.WithLibPath(libraryPath)); err != nil {
		return fmt.Errorf("agent: initialize Kronk: %w", err)
	}
	kronkInitialization.libraryPath = libraryPath
	return nil
}

func (generator *kronkGenerator) Generate(ctx context.Context, messages []Message, maxTokens int) (string, error) {
	conversation := make([]model.D, 0, len(messages))
	for _, message := range messages {
		if message.Role != "system" && message.Role != "user" && message.Role != "assistant" {
			continue
		}
		conversation = append(conversation, model.D{"role": message.Role, "content": message.Content})
	}
	if maxTokens < 1 || maxTokens > generator.maxTokens {
		maxTokens = generator.maxTokens
	}
	document := model.D{
		"messages":   conversation,
		"max_tokens": maxTokens,
	}
	stream, err := generator.client.ChatStreaming(ctx, document)
	if err != nil {
		return "", fmt.Errorf("Kronk chat request: %w", err)
	}
	var response strings.Builder
	for chunk := range stream {
		if len(chunk.Choices) == 0 {
			continue
		}
		choice := chunk.Choices[0]
		if choice.FinishReason() == model.FinishReasonError {
			message := "Kronk returned an inference error"
			if choice.Delta != nil && choice.Delta.Content != "" {
				message = choice.Delta.Content
			}
			return "", errors.New(message)
		}
		if choice.Delta != nil {
			response.WriteString(choice.Delta.Content)
		}
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if strings.TrimSpace(response.String()) == "" {
		return "", errors.New("Kronk returned an empty response")
	}
	return strings.TrimSpace(response.String()), nil
}

func (generator *kronkGenerator) Close(ctx context.Context) error {
	generator.once.Do(func() {
		generator.closeErr = generator.client.Unload(ctx)
	})
	return generator.closeErr
}
