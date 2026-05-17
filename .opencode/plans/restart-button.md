# Plan: Restart Button

## Goal
Add a "Restart" button to the header that triggers a full server restart (stop all models, reload config, restart).

## Changes

### 1. `proxy/proxymanager_api.go`
- Add `POST /api/restart` endpoint
- Handler emits `ConfigFileChangedEvent{ReloadingStateStart}` which triggers existing reload flow in `llama-swap.go`
- Return `200 OK` immediately

```go
apiGroup.POST("/restart", pm.apiRestart)

func (pm *ProxyManager) apiRestart(c *gin.Context) {
    event.Emit(ConfigFileChangedEvent{ReloadingState: ReloadingStateStart})
    c.JSON(http.StatusOK, gin.H{"msg": "ok"})
}
```

### 2. `ui-svelte/src/components/Header.svelte`
- Add "Restart" button to the left of "Playground" tab
- On click: `fetch('/api/restart', {method: 'POST'})` then `setTimeout(() => location.reload(), 4000)`
- Show brief "Restarting..." feedback

## Behavior
- Clicking Restart stops all models (SIGTERM → 10s grace → SIGKILL), reloads config, creates new ProxyManager
- Page shows "Restarting..." for ~4s then reloads via `location.reload()`
- SSE reconnects automatically on page load

## Testing
- `make test-dev`
- Manual: click button, verify models stop, server restarts, page reconnects
