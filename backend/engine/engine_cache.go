package engine

import (
	"sync"

	"github.com/google/uuid"
)

type EngineCache struct {
	mu      sync.RWMutex
	engines map[uuid.UUID]*Engine
}

func NewEngineCache() *EngineCache {
	return &EngineCache{
		engines: make(map[uuid.UUID]*Engine),
	}
}

func (c *EngineCache) Get(storyID uuid.UUID) (*Engine, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	eng, ok := c.engines[storyID]
	return eng, ok
}

func (c *EngineCache) Set(storyID uuid.UUID, eng *Engine) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.engines[storyID] = eng
}

func (c *EngineCache) Invalidate(storyID uuid.UUID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.engines, storyID)
}
