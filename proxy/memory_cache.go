package proxy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type MemoryCache struct {
	mu   sync.RWMutex
	data map[string]ProcessMemory
	path string
}

var defaultCache *MemoryCache

func init() {
	defaultCache = NewMemoryCache()
}

func getCachePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".cache", "llama-swap", "memory-cache.json")
}

func NewMemoryCache() *MemoryCache {
	mc := &MemoryCache{
		data: make(map[string]ProcessMemory),
		path: getCachePath(),
	}
	if mc.path != "" {
		mc.Load()
	}
	return mc
}

func (mc *MemoryCache) Load() {
	if mc.path == "" {
		return
	}
	data, err := os.ReadFile(mc.path)
	if err != nil {
		return
	}
	var cached map[string]ProcessMemory
	if err := json.Unmarshal(data, &cached); err != nil {
		return
	}
	mc.mu.Lock()
	mc.data = cached
	mc.mu.Unlock()
}

func (mc *MemoryCache) Set(id string, ram, vram int64) {
	mc.mu.Lock()
	mc.data[id] = ProcessMemory{RAMBytes: ram, VRAMBytes: vram}
	mc.mu.Unlock()
}

func (mc *MemoryCache) Get(id string) (ram, vram int64) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	if entry, ok := mc.data[id]; ok {
		return entry.RAMBytes, entry.VRAMBytes
	}
	return 0, 0
}

func (mc *MemoryCache) Save() {
	if mc.path == "" {
		return
	}
	mc.mu.RLock()
	data, err := json.MarshalIndent(mc.data, "", "  ")
	mc.mu.RUnlock()
	if err != nil {
		return
	}
	dir := filepath.Dir(mc.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	if err := os.WriteFile(mc.path, data, 0o644); err != nil {
		return
	}
}
