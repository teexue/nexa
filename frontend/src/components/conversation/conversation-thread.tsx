import { useState } from "react"
import type { ConversationEntry } from "@/types/agent"
import { isTurnConclusion } from "@/lib/activity-fold"
import { ActivityEntry } from "./activity-entry"
import { FileChangeSummary } from "./workspace-file-changes"

/** Read-only conversation transcript: same bubbles, tools, and file summary as chat. */
export function ConversationThread({
  messages,
  isStreaming,
}: {
  messages: ConversationEntry[]
  isStreaming?: boolean
}) {
  const [selectedToolCallId, setSelectedToolCallId] = useState<string | null>(
    null
  )
  return (
    <div className="flex flex-col gap-2.5">
      {messages.map((entry, i) => (
        <ActivityEntry
          key={entry.id}
          entry={entry}
          selectedToolCallId={selectedToolCallId}
          onSelectToolCall={setSelectedToolCallId}
          isActive={!!isStreaming && i === messages.length - 1}
          isConclusion={isTurnConclusion(messages, i, !!isStreaming)}
        />
      ))}
      <FileChangeSummary messages={messages} className="mx-0 mt-1" />
    </div>
  )
}
