export interface FoldEntry {
  id: string
  role: string
  content: string
}

export function latestTextLine(text: string): string {
  return pickLine(text, "last")
}

export function firstTextLine(text: string): string {
  return pickLine(text, "first")
}

function pickLine(text: string, which: "first" | "last"): string {
  const lines = text
    .split("\n")
    .map((line) => line.trim())
    .filter(Boolean)
  const raw = which === "last" ? lines.at(-1) : lines[0]
  if (!raw) return ""
  return raw.replace(/^#{1,6}\s+/, "").replace(/^[-*]\s+/, "")
}

export function isTurnConclusion(
  messages: FoldEntry[],
  index: number,
  streaming: boolean
): boolean {
  const entry = messages[index]
  if (!entry || entry.role !== "assistant" || !entry.content) return false
  const start = turnStart(messages, index)
  const end = turnEnd(messages, start)
  const last = lastContentIndex(messages, start, end)
  if (last !== index) return false
  const turnIsLive = streaming && end === messages.length
  return !turnIsLive || index === messages.length - 1
}

function turnStart(messages: FoldEntry[], index: number): number {
  for (let i = index; i >= 0; i--) {
    if (messages[i].role === "user") return i + 1
  }
  return 0
}

function turnEnd(messages: FoldEntry[], start: number): number {
  for (let i = start; i < messages.length; i++) {
    if (messages[i].role === "user") return i
  }
  return messages.length
}

function lastContentIndex(
  messages: FoldEntry[],
  start: number,
  end: number
): number {
  let last = -1
  for (let i = start; i < end; i++) {
    if (messages[i].role === "assistant" && messages[i].content) last = i
  }
  return last
}
