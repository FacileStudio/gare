package engine

import (
	"context"
	"fmt"
)

// AddTags adds tags to an app and returns its tags afterwards.
func (e *Engine) AddTags(ctx context.Context, name string, tags []string) ([]string, error) {
	unlock, cfg, err := e.lockAndLoad(ctx, name)
	if err != nil {
		return nil, err
	}
	defer unlock()
	if err := cfg.AddTags(tags...); err != nil {
		return nil, err
	}
	if err := e.store.Save(cfg); err != nil {
		return nil, fmt.Errorf("failed to save config: %w", err)
	}
	return cfg.Tags, nil
}

// RemoveTags removes tags from an app and returns its tags afterwards, and whether any of the
// given tags was there to remove. Nothing is saved when none was.
func (e *Engine) RemoveTags(ctx context.Context, name string, tags []string) ([]string, bool, error) {
	unlock, cfg, err := e.lockAndLoad(ctx, name)
	if err != nil {
		return nil, false, err
	}
	defer unlock()
	if !cfg.RemoveTags(tags...) {
		return cfg.Tags, false, nil
	}
	if err := e.store.Save(cfg); err != nil {
		return nil, false, fmt.Errorf("failed to save config: %w", err)
	}
	return cfg.Tags, true, nil
}
