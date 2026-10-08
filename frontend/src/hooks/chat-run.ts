import { apiHeaders } from "@/lib/api"
import type { ChatAction } from "./use-chat-state"
import { parseSSELine } from "./use-chat-messages"
import { dispatchSSEEvent } from "./chat-sse"

export async function processSSEStream(
  reader: ReadableStreamDefaultReader<Uint8Array>,
  entryId: string,
  dispatch: (action: ChatAction) => void
): Promise<void> {
  const decoder = new TextDecoder()
  let buffer = ""
  while (true) {
    const { done, value } = await reader.read()
    if (done) break
    buffer += decoder.decode(value, { stream: true })
    const lines = buffer.split("\n")
    buffer = lines.pop() ?? ""
    for (const line of lines) {
      const trimmed = line.trim()
      if (!trimmed) continue
      const event = parseSSELine(trimmed)
      if (!event) continue
      if (dispatchSSEEvent(event, entryId, dispatch)) return
    }
  }
  dispatch({ type: "STREAM_DONE", entryId, status: "completed", turns: 0 })
}

export interface SendRunOpts {
  agent: string
  prompt: string
  sessionId: string | null
  workDir: string | undefined
  signal: AbortSignal
  entryId: string
  dispatch: (action: ChatAction) => void
  images?: { dataUrl: string; name: string }[]
  model?: string
  provider?: string
  thinkingEffort?: string
}

function buildRunBody(opts: SendRunOpts): Record<string, unknown> {
  const body: Record<string, unknown> = {
    agent: opts.agent,
    prompt: opts.prompt,
  }
  if (opts.sessionId) body.session_id = opts.sessionId
  if (opts.workDir) body.workdir = opts.workDir
  if (opts.model) body.model = opts.model
  if (opts.provider) body.provider = opts.provider
  if (opts.thinkingEffort) body.thinking_effort = opts.thinkingEffort
  if (opts.images && opts.images.length > 0) {
    body.images = opts.images.map((img) => ({
      data_url: img.dataUrl,
      name: img.name,
    }))
  }
  return body
}

export async function sendRunRequest(opts: SendRunOpts): Promise<void> {
  // Backend rebuilds history from the session so the prompt prefix stays cache-stable.
  const res = await fetch("/v1/agents/run", {
    method: "POST",
    headers: { "Content-Type": "application/json", ...apiHeaders() },
    body: JSON.stringify(buildRunBody(opts)),
    signal: opts.signal,
  })
  if (!res.ok) {
    const errBody = await res.json().catch(() => null)
    throw new Error(errBody?.message ?? `HTTP ${res.status}`)
  }
  // Session id is in a response header before the first SSE frame so the URL
  // can be updated immediately and survive a page refresh.
  const headerSessionId = res.headers.get("x-session-id")
  if (headerSessionId) {
    opts.dispatch({ type: "SET_SESSION_ID", sessionId: headerSessionId })
  }
  const reader = res.body?.getReader()
  if (!reader) throw new Error("No response body")
  await processSSEStream(reader, opts.entryId, opts.dispatch)
}
