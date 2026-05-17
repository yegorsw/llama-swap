# Model Memory: Delayed Refresh + Cached Values — ✅ COMPLETED

## Overview

Two enhancements to the RAM/VRAM columns in the Models tab:

1. **Delayed refresh**: 5 seconds after a model transitions to `ready`, re-read memory and push an SSE update to catch memory that settles after initial load
2. **Cached display**: When a model is stopped, show the last-known memory values in a darker grey color

## Implementation — All Done

| Change | File | Status |
|--------|------|--------|
| Cached memory fields + methods on Process | `proxy/process.go` | ✅ |
| `MemoryStale` field on Model struct | `proxy/proxymanager_api.go:27` | ✅ |
| `getModelStatus()` caches on ready, returns stale on stopped | `proxy/proxymanager_api.go:93-107` | ✅ |
| 5s delayed SSE refresh after `StateReady` | `proxy/proxymanager_api.go:220-232` | ✅ |
| `memoryStale` in TypeScript Model interface | `ui-svelte/src/lib/types.ts` | ✅ |
| Stale values styled with `opacity-50` | `ui-svelte/src/components/ModelsPanel.svelte:192-193` | ✅ |

## Behavior

| Model State | RAM/VRAM Display | Color |
|---|---|---|
| `ready` (initial) | Current memory read | Normal |
| `ready` (~5s later) | Refreshed memory | Normal |
| `stopped` (was loaded) | Last known values | Dark grey, faded |
| `stopped` (never loaded) | `-` | Normal |

## Notes

- The 5s delayed refresh fires per-model on each `ready` transition
- Cached memory persists in-process (lost on llama-swap restart)
- The SSE buffer has capacity 25, so the delayed refresh won't block if the buffer is full (uses `default:` case)
