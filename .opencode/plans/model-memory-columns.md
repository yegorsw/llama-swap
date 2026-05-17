# Model Memory Usage (RAM + VRAM) Columns

## Overview

Add two new columns ("RAM" and "VRAM") to the Models tab that show the actual in-memory usage of each loaded llama-server process. RAM is read from `/proc/<pid>/status` (VmRSS), VRAM from `nvidia-smi --query-compute-apps`.

## Implementation

### 1. New file: `proxy/process_memory.go`

New file with a `ProcessMemory` struct and helper functions:

```go
package proxy

import (
    "bufio"
    "fmt"
    "os"
    "os/exec"
    "strconv"
    "strings"
)

type ProcessMemory struct {
    RAMBytes int64
    VRAMBytes int64
}

func (p *Process) GetMemoryUsage() (ProcessMemory, error) {
    var result ProcessMemory

    if p.cmd == nil || p.cmd.Process == nil {
        return result, nil
    }

    pid := p.cmd.Process.Pid

    // Read RAM from /proc/<pid>/status
    ram, err := getProcessRSS(pid)
    if err != nil {
        p.proxyLogger.Debugf("<%s> could not read RSS: %v", p.ID, err)
    } else {
        result.RAMBytes = ram
    }

    // Read VRAM from nvidia-smi
    vram, err := getProcessVRAM(pid)
    if err != nil {
        p.proxyLogger.Debugf("<%s> could not read VRAM: %v", p.ID, err)
    } else {
        result.VRAMBytes = vram
    }

    return result, nil
}

func getProcessRSS(pid int) (int64, error) {
    file, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
    if err != nil {
        return 0, err
    }

    scanner := bufio.NewScanner(strings.NewReader(string(file)))
    for scanner.Scan() {
        line := scanner.Text()
        if strings.HasPrefix(line, "VmRSS:") {
            parts := strings.Fields(strings.TrimPrefix(line, "VmRSS:"))
            if len(parts) >= 2 {
                kb, err := strconv.ParseInt(parts[0], 10, 64)
                if err != nil {
                    return 0, err
                }
                return kb * 1024, nil // convert kB to bytes
            }
        }
    }
    return 0, fmt.Errorf("VmRSS not found")
}

func getProcessVRAM(pid int) (int64, error) {
    cmd := exec.Command("nvidia-smi",
        "--query-compute-apps=pid,used_memory",
        "--format=csv,noheader,nounits")
    output, err := cmd.Output()
    if err != nil {
        return 0, err
    }

    pidStr := fmt.Sprintf("%d", pid)
    scanner := bufio.NewScanner(strings.NewReader(string(output)))
    for scanner.Scan() {
        line := scanner.Text()
        parts := strings.SplitN(line, ",", 2)
        if len(parts) == 2 && strings.TrimSpace(parts[0]) == pidStr {
            memStr := strings.TrimSpace(parts[1])
            // nvidia-smi returns "XXX MiB" or just "XXX" with nounits
            memStr = strings.TrimSpace(strings.TrimSuffix(memStr, " MiB"))
            miB, err := strconv.ParseInt(memStr, 10, 64)
            if err != nil {
                continue
            }
            return miB * 1024 * 1024, nil // convert MiB to bytes
        }
    }
    return 0, fmt.Errorf("pid %d not found in nvidia-smi output", pid)
}
```

### 2. Modify: `proxy/proxymanager_api.go`

**Add fields to Model struct** (line 16-24):
```go
type Model struct {
    Id          string   `json:"id"`
    Name        string   `json:"name"`
    Description string   `json:"description"`
    State       string   `json:"state"`
    Unlisted    bool     `json:"unlisted"`
    PeerID      string   `json:"peerID"`
    Aliases     []string `json:"aliases,omitempty"`
    RAMBytes    int64    `json:"ramBytes,omitempty"`
    VRAMBytes   int64    `json:"vramBytes,omitempty"`
}
```

**Populate in `getModelStatus()`** (after line 81, before line 82 where `models = append`):

After the state is determined from `process.CurrentState()`, add:
```go
        var ramBytes int64
        var vramBytes int64
        if process != nil && process.CurrentState() == StateReady {
            if mem, err := process.GetMemoryUsage(); err == nil {
                ramBytes = mem.RAMBytes
                vramBytes = mem.VRAMBytes
            }
        }
```

Then in the `models = append(models, Model{...})` call (line 82-89), add:
```go
            RAMBytes:  ramBytes,
            VRAMBytes: vramBytes,
```

### 3. Modify: `ui-svelte/src/lib/types.ts`

Add to `Model` interface (after line 12):
```typescript
  ramBytes?: number;
  vramBytes?: number;
```

### 4. Modify: `ui-svelte/src/components/ModelsPanel.svelte`

**Add formatBytes helper** (in `<script>`, after line 55):
```typescript
  function formatBytes(bytes?: number): string {
    if (!bytes || bytes === 0) return "-";
    if (bytes < 1024 * 1024) {
      return `${(bytes / 1024).toFixed(0)} KB`;
    }
    if (bytes < 1024 * 1024 * 1024) {
      return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
    }
    return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`;
  }
```

**Add column headers** (lines 152-156, replace the `<tr>`):
```svelte
        <tr class="text-left border-b border-gray-200 dark:border-white/10 bg-surface">
          <th>{$showIdorNameStore === "id" ? "Model ID" : "Name"}</th>
          <th></th>
          <th>RAM</th>
          <th>VRAM</th>
          <th>State</th>
        </tr>
```

**Add data cells** (lines 159-183, add two new `<td>` cells before the State cell):
```svelte
            <td class="w-20 text-right font-mono text-sm">{formatBytes(model.ramBytes)}</td>
            <td class="w-20 text-right font-mono text-sm">{formatBytes(model.vramBytes)}</td>
            <td class="w-20">
              <span class="w-16 text-center status status--{model.state}">{model.state}</span>
            </td>
```

## Notes

- VRAM only populates for `ready` state (process is running). Stopped models show `-`.
- If `nvidia-smi` is not available, VRAM shows `-` (error silently logged on backend).
- Values update on SSE events (model load/unload state changes), not continuously polling.
- No new Go dependencies required.
