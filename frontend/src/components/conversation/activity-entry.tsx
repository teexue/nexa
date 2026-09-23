import { useState } from "react"
import { useTranslation } from "react-i18next"
import { ChevronDown, ChevronRight, Minimize2 } from "lucide-react"
import { MarkdownRenderer } from "@/components/shared/markdown-renderer"
import { ThinkingBlock } from "./thinking-block"
import { ToolCallGroup } from "./tool-call-group"
import { UserMessage } from "./user-message"
import type { ConversationEntry } from "@/types/agent"

interface ActivityEntryProps {
  entry: ConversationEntry
  selectedToolCallId: string | null
  onSelectToolCall: (id: string) => void
  onApproveTool?: (approvalId: string) => void
  onDenyTool?: (approvalId: string) => void
  isActive?: boolean
}

function GeneratingPulse() {
  const { t } = useTranslation()
  return (
    <span className="flex items-center gap-1 text-[11px] text-primary">
      <span className="h-1.5 w-1.5 animate-pulse rounded-full bg-primary" />
      {t("status.generating")}
    </span>
  )
}

export function ActivityEntry(props: ActivityEntryProps) {
  const { entry, isActive } = props
  const [thinkingExpanded, setThinkingExpanded] = useState(false)
  if (entry.compactionSummary)
    return <CompactionBanner summary={entry.compactionSummary} />
  if (entry.role === "user") return <UserMessage entry={entry} />
  return (
    <AssistantBody
      props={props}
      thinkingExpanded={thinkingExpanded}
      onToggleThinking={() => setThinkingExpanded((v) => !v)}
      isWaiting={
        !!isActive &&
        !entry.content &&
        !entry.reasoningContent &&
        !(entry.toolCalls && entry.toolCalls.length > 0)
      }
    />
  )
}

function AssistantBody({
  props,
  thinkingExpanded,
  onToggleThinking,
  isWaiting,
}: {
  props: ActivityEntryProps
  thinkingExpanded: boolean
  onToggleThinking: () => void
  isWaiting: boolean
}) {
  const { entry, isActive } = props
  const hasToolCalls = entry.toolCalls && entry.toolCalls.length > 0
  return (
    <div className="flex min-w-0 flex-col gap-1.5">
      {isActive && <GeneratingPulse />}
      {entry.reasoningContent && (
        <ThinkingBlock
          content={entry.reasoningContent}
          isStreaming={!!isActive}
          isExpanded={thinkingExpanded}
          onToggle={onToggleThinking}
        />
      )}
      {hasToolCalls && (
        <ToolCallGroup
          toolCalls={entry.toolCalls!}
          selectedToolCallId={props.selectedToolCallId}
          onSelectToolCall={props.onSelectToolCall}
          onApproveTool={props.onApproveTool}
          onDenyTool={props.onDenyTool}
        />
      )}
      {entry.content && (
        <div className="min-w-0 text-[13px] leading-relaxed">
          <MarkdownRenderer
            content={entry.content}
            isStreaming={!!isActive && !hasToolCalls}
          />
        </div>
      )}
      {entry.outputTruncated && !isActive && <OutputTruncatedBanner />}
      {isWaiting && (
        <div className="flex flex-col gap-1.5" aria-hidden>
          <div className="shimmer-line" />
          <div className="shimmer-line" />
          <div className="shimmer-line" />
        </div>
      )}
    </div>
  )
}

function OutputTruncatedBanner() {
  const { t } = useTranslation()
  return (
    <div className="rounded-lg border border-warning/30 bg-warning/10 px-3 py-2">
      <p className="text-xs leading-relaxed text-warning">
        {t("conversation.outputTruncated")}
      </p>
    </div>
  )
}

function CompactionBanner({ summary }: { summary: string }) {
  const { t } = useTranslation()
  const [expanded, setExpanded] = useState(false)
  return (
    <div className="rounded-lg border border-warning/30 bg-warning/10">
      <button
        onClick={() => setExpanded((v) => !v)}
        className="flex w-full items-center gap-2 px-3 py-2 text-left"
      >
        <Minimize2 className="h-3.5 w-3.5 shrink-0 text-warning" />
        <span className="flex-1 text-xs font-medium text-warning">
          {t("conversation.compaction")}
        </span>
        {expanded ? (
          <ChevronDown className="h-3 w-3 text-warning/70" />
        ) : (
          <ChevronRight className="h-3 w-3 text-warning/70" />
        )}
      </button>
      {expanded && (
        <div className="border-t border-warning/20 px-3 py-2">
          <p className="text-xs leading-relaxed whitespace-pre-wrap text-warning/80">
            {summary}
          </p>
        </div>
      )}
    </div>
  )
}
