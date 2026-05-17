<script lang="ts">
  import type { ReqRespCapture } from "../lib/types";
  import type { ChatMessage as ChatMessageType } from "../lib/types";
  import { getTextContent } from "../lib/types";
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

  let messages = $derived.by((): ChatMessageType[] => {
    if (!capture) return [];

    const reqBody = decodeBody(capture.req_body);
    if (!reqBody) return [];

    let reqJson: { messages?: ChatMessageType[] } = {};
    try {
      reqJson = JSON.parse(reqBody);
    } catch {
      return [];
    }

    const convMessages: ChatMessageType[] = [];
    for (const m of reqJson.messages || []) {
      if (m.role === "user" || m.role === "assistant" || m.role === "system") {
        convMessages.push({
          role: m.role,
          content: m.content || "",
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
            {#if msg.role === "system"}
              <div class="flex justify-center mb-2">
                <div class="rounded px-3 py-1.5 text-xs font-mono text-txtsecondary bg-secondary border border-card-border max-w-[85%]">
                  <span class="font-semibold text-txtsecondary">system:</span> {getTextContent(msg.content)}
                </div>
              </div>
            {:else}
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
