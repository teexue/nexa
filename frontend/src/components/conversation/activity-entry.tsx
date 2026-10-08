import { useState } from "react"
import { useTranslation } from "react-i18next"
import { ChevronDown, ChevronRight, Minimize2 } from "lucide-react"
import { MarkdownRenderer } from "@/components/shared/markdown-renderer"
import { firstTextLine } from "@/lib/activity-fold"
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
  isConclusion?: boolean
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
  if (entry.compactionSummary)
    return <CompactionBanner summary={entry.compactionSummary} />
  if (entry.role === "user") return <UserMessage entry={entry} />
  return (
    <AssistantBody
      props={props}
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
  isWaiting,
}: {
  props: ActivityEntryProps
  isWaiting: boolean
}) {
  return (
    <div className="flex min-w-0 flex-col gap-1.5">
      {props.isActive ? <GeneratingPulse /> : null}
      <ThinkingSlot props={props} />
      <ToolSlot props={props} />
      <ReplySlot props={props} />
      <TruncationNote entry={props.entry} isActive={!!props.isActive} />
      {isWaiting ? <WaitingShimmer /> : null}
    </div>
  )
}

function ThinkingSlot({ props }: { props: ActivityEntryProps }) {
  const { entry, isActive } = props
  if (!entry.reasoningContent) return null
  const busy = !!isActive && !entry.content && !entry.toolCalls?.length
  return (
    <ThinkingBlock
      key={isActive ? "live" : "done"}
      content={entry.reasoningContent}
      isStreaming={busy}
    />
  )
}

function ToolSlot({ props }: { props: ActivityEntryProps }) {
  const calls = props.entry.toolCalls
  if (!calls?.length) return null
  const lastTool = calls[calls.length - 1]?.id ?? ""
  const phase = props.isActive ? "live" : "done"
  return (
    <ToolCallGroup
      toolCalls={calls}
      foldKey={`${phase}:${lastTool}`}
      selectedToolCallId={props.selectedToolCallId}
      onSelectToolCall={props.onSelectToolCall}
      onApproveTool={props.onApproveTool}
      onDenyTool={props.onDenyTool}
    />
  )
}

function ReplySlot({ props }: { props: ActivityEntryProps }) {
  const { entry, isActive, isConclusion } = props
  if (!entry.content) return null
  const open = !!isConclusion || (!!isActive && !!entry.content)
  return (
    <ReplyBlock
      key={open ? "open" : "shut"}
      content={entry.content}
      initiallyOpen={open}
      pinned={!!isConclusion}
      streaming={!!isActive && !entry.toolCalls?.length}
    />
  )
}

function TruncationNote({
  entry,
  isActive,
}: {
  entry: ConversationEntry
  isActive: boolean
}) {
  if (!entry.outputTruncated || isActive) return null
  return <OutputTruncatedBanner />
}

function WaitingShimmer() {
  return (
    <div className="flex flex-col gap-1.5" aria-hidden>
      <div className="shimmer-line" />
      <div className="shimmer-line" />
      <div className="shimmer-line" />
    </div>
  )
}

function ReplyBlock({
  content,
  initiallyOpen,
  pinned,
  streaming,
}: {
  content: string
  initiallyOpen: boolean
  pinned: boolean
  streaming: boolean
}) {
  const [open, setOpen] = useState(initiallyOpen)
  const line = firstTextLine(content)
  if (!open) {
    return (
      <button
        type="button"
        onClick={() => setOpen(true)}
        className="flex w-full min-w-0 items-center gap-1 text-left text-[13px] text-muted-foreground"
      >
        <ChevronRight className="h-3 w-3 shrink-0" />
        <span className="truncate">{line}</span>
      </button>
    )
  }
  return (
    <div className="min-w-0 text-[13px] leading-relaxed">
      {!pinned && (
        <button
          type="button"
          onClick={() => setOpen(false)}
          className="mb-1 flex w-full min-w-0 items-center gap-1 text-left text-[11px] text-muted-foreground"
        >
          <ChevronDown className="h-3 w-3 shrink-0" />
          <span className="truncate">{line}</span>
        </button>
      )}
      <MarkdownRenderer content={content} isStreaming={streaming} />
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
