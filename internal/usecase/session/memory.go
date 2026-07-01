package session

import "sync"

// MemoryModelStore keeps per-session selected models in memory.
type MemoryModelStore struct {
	defaultModel string
	models       map[string]string
	mu           sync.RWMutex
}

func NewMemoryModelStore(defaultModel string) *MemoryModelStore {
	return &MemoryModelStore{
		defaultModel: defaultModel,
		models:       make(map[string]string),
	}
}

func (s *MemoryModelStore) SetModel(sessionID string, model string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.models[sessionID] = model
}

func (s *MemoryModelStore) GetModel(sessionID string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if model, ok := s.models[sessionID]; ok && model != "" {
		return model
	}
	return s.defaultModel
}
