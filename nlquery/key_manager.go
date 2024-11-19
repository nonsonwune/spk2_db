package nlquery

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"
)

// KeyManager handles API key rotation
type KeyManager struct {
	keys     []string
	current  uint32
	mu       sync.RWMutex
}

// NewKeyManager creates a new key manager with available API keys
func NewKeyManager() *KeyManager {
	keys := make([]string, 0)
	
	// Load all available API keys
	for i := 1; i <= 4; i++ {
		key := os.Getenv(fmt.Sprintf("GEMINI_API_KEY_%d", i))
		if key != "" {
			keys = append(keys, key)
		}
	}
	
	return &KeyManager{
		keys:    keys,
		current: 0,
	}
}

// GetNextKey returns the next API key in rotation
func (km *KeyManager) GetNextKey() string {
	km.mu.RLock()
	defer km.mu.RUnlock()
	
	if len(km.keys) == 0 {
		return ""
	}
	
	// Atomically increment and wrap around
	current := atomic.AddUint32(&km.current, 1)
	index := (current - 1) % uint32(len(km.keys))
	
	return km.keys[index]
}

// MarkKeyFailed can be used to temporarily disable a key that's failing
// This is a placeholder for future implementation of key health tracking
func (km *KeyManager) MarkKeyFailed(key string) {
	// TODO: Implement key health tracking and temporary disablement
	// For now, we just continue rotating
}
