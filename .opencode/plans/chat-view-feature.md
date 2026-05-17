# Chat View Feature - Implementation Plan

## Files to Modify

### 1. `ui-svelte/src/routes/Activity.svelte`

#### Add import (line 1, after existing imports):
```typescript
import ChatViewDialog from "../components/ChatViewDialog.svelte";
```

#### Add "chat" to ColumnKey type (line 10-23):
Add `"chat"` to the union type, e.g. after `"capture"`:
```typescript
type ColumnKey =
  | "id"
  | "time"
  // ... existing keys ...
  | "capture"
  | "chat";
```

#### Add column definition (line 31-45):
Add to the `columns` array after the capture column:
```typescript
{ key: "chat", label: "Chat", defaultVisible: true },
```

#### Add state variables (after line 126):
```typescript
let chatCapture = $state<ReqRespCapture | null>(null);
let chatDialogOpen = $state(false);
let loadingChatId = $state<number | null>(null);
```

#### Add handler functions (after line 141):
```typescript
async function openChatView(id: number) {
  loadingChatId = id;
  const capture = await getCapture(id);
  loadingChatId = null;
  if (capture) {
    chatCapture = capture;
    chatDialogOpen = true;
  }
}

function closeChatDialog() {
  chatDialogOpen = false;
  chatCapture = null;
}
```

#### Add table header (after line 229, the Capture `<th>`):
```svelte
{#if $visibleColumns.includes("chat")}
  <th class="px-6 py-3">Chat</th>
{/if}
```

#### Add table cell (after line 292, the Capture `<td>` block):
```svelte
{#if $visibleColumns.includes("chat")}
  <td class="px-6 py-4">
    {#if metric.has_capture}
      <button
        onclick={() => openChatView(metric.id)}
        disabled={loadingChatId === metric.id}
        class="btn btn--sm"
      >
        {loadingChatId === metric.id ? "..." : "Chat"}
      </button>
    {:else}
      <span class="text-txtsecondary">-</span>
    {/if}
  </td>
{/if}
```

#### Add dialog component (after line 301, the CaptureDialog):
```svelte
<ChatViewDialog capture={chatCapture} open={chatDialogOpen} onclose={closeChatDialog} />
```

---

### 2. `ui-svelte/src/components/ChatViewDialog.svelte` (NEW FILE)

```svelte
<script lang="ts">
  import type { ReqRespCapture, ChatMessage } from "../lib/types";
  import ChatMessage from "./playground/ChatMessage.svelte";

  interface Props {
    capture: ReqRespCapture | null;
    open: boolean;
    onclose: () => void;
  }

  let { capture, open, onclose }: Props = $props();
  let dialogEl: HTMLDialogElement | undefined = $state();

  $effect(() => {
    if (open && dialogEl) {
      dialogEl.showModal();
    } else if (!open && dialogEl) {
      dialogEl.close();
    }
  });

  function handleDialogClose() {
    onclose();
  }

  function decodeBody(body: string | null | undefined): string {
    if (!body) return "";
    try {
      const binary = atob(body);
      const bytes = Uint8Array.from(binary, (c) => c.charCodeAt(0));
      return new TextDecoder().decode(bytes);
    } catch {
      return body;
    }
  }

  function getContentType(
    headers: Record<string, string> | null | undefined,
  ): string {
    if (!headers) return "";
    const ct = headers["Content-Type"] || headers["content-type"] || "";
    return ct.toLowerCase();
  }

  interface ParsedAssistantMessage {
    content: string;
    reasoning_content: string;
  }

  function parseResponseAssistantMessage(
    respBody: string,
    respContentType: string,
  ): ParsedAssistantMessage {
    const result: ParsedAssistantMessage = { content: "", reasoning_content: "" };

    if (respContentType.includes("text/event-stream")) {
      for (const line of respBody.split("\n")) {
        const trimmed = line.trim();
        if (!trimmed || !trimmed.startsWith("data: ")) continue;
        const data = trimmed.slice(6);
        if (data === "[DONE]") continue;
        try {
          const parsed = JSON.parse(data);
          const delta = parsed.choices?.[0]?.delta;
          if (delta?.content) result.content += delta.content;
          if (delta?.reasoning_content) result.reasoning_content += delta.reasoning_content;
          if (delta?.reasoning) result.reasoning_content += delta.reasoning;
        } catch {
          // skip
        }
      }
    } else if (respContentType.includes("json")) {
      try {
        const parsed = JSON.parse(respBody);
        const msg = parsed.choices?.[0]?.message;
        if (msg) {
          result.content = msg.content || "";
          result.reasoning_content = msg.reasoning_content || msg.reasoning || "";
        }
      } catch {
        // skip
      }
    }

    return result;
  }

  let messages = $derived.by((): ChatMessage[] => {
    if (!capture) return [];

    const reqBody = decodeBody(capture.req_body);
    if (!reqBody) return [];

    let reqJson: { messages?: ChatMessage[] } = {};
    try {
      reqJson = JSON.parse(reqBody);
    } catch {
      return [];
    }

    const convMessages: ChatMessage[] = [];
    for (const m of reqJson.messages || []) {
      if (m.role === "user" || m.role === "assistant" || m.role === "system") {
        convMessages.push({
          role: m.role,
          content: typeof m.content === "string" ? m.content : m.content,
          reasoning_content:
            m.role === "assistant" ? m.reasoning_content : undefined,
        });
      }
    }

    const respBody = decodeBody(capture.resp_body);
    const respCt = getContentType(capture.resp_headers);
    const assistantMsg = parseResponseAssistantMessage(respBody, respCt);

    if (assistantMsg.content || assistantMsg.reasoning_content) {
      convMessages.push({
        role: "assistant",
        content: assistantMsg.content,
        reasoning_content: assistantMsg.reasoning_content || undefined,
      });
    }

    return convMessages;
  });

  let modelName = $derived.by((): string => {
    if (!capture) return "";
    const reqBody = decodeBody(capture.req_body);
    try {
      const parsed = JSON.parse(reqBody);
      return parsed.model || "";
    } catch {
      return "";
    }
  });
</script>

<dialog
  bind:this={dialogEl}
  onclose={handleDialogClose}
  class="bg-background text-txtmain rounded-lg shadow-xl max-w-3xl w-full max-h-[90vh] p-0 backdrop:bg-black/50 m-auto"
>
  {#if capture}
    <div class="flex flex-col max-h-[90vh]">
      <div class="flex justify-between items-center p-4 border-b border-card-border">
        <div>
          <h2 class="text-lg font-bold">Chat View #{capture.id + 1}</h2>
          {#if modelName}
            <p class="text-xs text-txtsecondary mt-0.5">{modelName}</p>
          {/if}
        </div>
        <button
          onclick={() => dialogEl?.close()}
          class="text-txtsecondary hover:text-txtmain text-2xl leading-none"
        >
          &times;
        </button>
      </div>

      <div class="overflow-y-auto flex-1 p-4 space-y-2">
        {#if messages.length === 0}
          <div class="text-center text-txtsecondary py-8">No messages to display</div>
        {:else}
          {#each messages as msg, idx (idx)}
            {#if msg.role !== "system"}
              <ChatMessage
                role={msg.role}
                content={msg.content}
                reasoning_content={msg.reasoning_content}
                isStreaming={false}
              />
            {/if}
          {/each}
        {/if}
      </div>

      <div class="p-4 border-t border-card-border flex justify-end">
        <button onclick={() => dialogEl?.close()} class="btn">Close</button>
      </div>
    </div>
  {/if}
</dialog>
```

## Summary of Changes

1. **Activity.svelte** - Add "chat" column with toggle visibility, "Chat" button per row, state management for dialog
2. **ChatViewDialog.svelte** (new) - Modal dialog that:
   - Decodes base64 request body to extract `messages` array
   - Decodes base64 response body (SSE or JSON) to get assistant's reply
   - Appends assistant reply as final message
   - Renders conversation using existing `ChatMessage.svelte` component with markdown, code highlighting, reasoning support
   - Hides system messages from display
