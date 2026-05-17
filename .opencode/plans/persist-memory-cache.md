# Plan: Persist Memory Cache to XDG Cache Directory

## Goal
Persist model memory cache to `~/.cache/llama-swap/memory-cache.json` so RAM/VRAM values survive server restarts.

## Approach: Package-level global `defaultCache` in `proxy` package. No constructor changes needed.

## Cache Format
```json
{
  "gemma-4-31B-it-Q4": {"ram": 12345678, "vram": 98765432},
  "qwen3.6-27B-Q8": {"ram": 23456789, "vram": 87654321}
}
```

## Implementation

### 1. New file: `proxy/memory_cache.go`
- `MemoryCache` struct with `sync.RWMutex` + `map[string]ProcessMemory`
- Package-level `var defaultCache *MemoryCache`
- `func init()` — creates `defaultCache = NewMemoryCache()`, loads from disk
- `Get(modelID)` / `Set(modelID, ram, vram)` — mutex-protected access
- `Save()` — writes map to JSON, creates `~/.cache/llama-swap/` dir if needed
- Path: `filepath.Join(os.UserHomeDir(), ".cache", "llama-swap", "memory-cache.json")`

### 2. `proxy/process.go` — Update existing methods (no struct/constructor changes)
- `SetLastMemory()` — after setting in-memory values, also calls `defaultCache.Set()` + `defaultCache.Save()`
- `GetLastMemory()` — if in-memory values are both 0, falls back to `defaultCache.Get()`

## Behavior
- On startup: `init()` loads cache from disk
- On model ready: in-memory cache updated, disk cache written
- On model stopped: returns cached value (in-memory first, then disk cache)
- Cache file is small (<1KB for 10 models), writes are infrequent
