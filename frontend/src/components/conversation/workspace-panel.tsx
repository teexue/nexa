import { useCallback, useState } from "react"
import { InputBar } from "./input-bar"
import type { SessionTokenUsage } from "./token-usage-indicator"
import { useAutoScroll } from "@/hooks/use-auto-scroll"
import type { MessageSearch } from "@/hooks/use-message-search"
import { optimizePrompt } from "@/lib/api"
import type { ConversationEntry, FileAttachment } from "@/types/agent"
import { ApprovalBar } from "./workspace-approval-bar"
import { WorkspaceEmptyState } from "./workspace-empty-state"
import { FileChangeSummary } from "./workspace-file-changes"
import { WorkspaceMessageList } from "./workspace-message-list"

interface WorkspacePanelProps {
  messages: ConversationEntry[]
  isStreaming: boolean
  error: string | null
  onSendMessage: (text: string, attachments: FileAttachment[]) => void
  onStop?: () => void
  selectedToolCallId: string | null
  onSelectToolCall: (id: string) => void
  onApproveTool?: (approvalId: string) => void
  onDenyTool?: (approvalId: string) => void
  noAgent?: boolean
  onCreateAgent?: () => void
  agentName?: string
  visionEnabled?: boolean
  search: MessageSearch
  inputAccessory?: React.ReactNode
  tokenUsage?: SessionTokenUsage
  sessionId?: string | null
}

function usePromptOptimize(agentName: string) {
  const [optimizing, setOptimizing] = useState(false)
  const onOptimize = useCallback(
    async (text: string) => {
      setOptimizing(true)
      try {
        const result = await optimizePrompt(text, { agent: agentName })
        return result.optimized_prompt
      } finally {
        setOptimizing(false)
      }
    },
    [agentName]
  )
  return { optimizing, onOptimize }
}

export function WorkspacePanel(props: WorkspacePanelProps) {
  const agentName = props.agentName ?? "agent"
  const { containerRef, handleScroll } = useAutoScroll(props.messages, {
    behavior: props.isStreaming ? "smooth" : "auto",
    resetKey: props.sessionId,
  })
  const optimize = usePromptOptimize(agentName)
  return (
    <WorkspaceColumn
      props={props}
      containerRef={containerRef}
      onScroll={handleScroll}
      optimize={optimize}
    />
  )
}

function WorkspaceColumn({
  props,
  containerRef,
  onScroll,
  optimize,
}: {
  props: WorkspacePanelProps
  containerRef: React.RefObject<HTMLDivElement | null>
  onScroll: () => void
  optimize: ReturnType<typeof usePromptOptimize>
}) {
  const isEmpty = props.messages.length === 0 && !props.error
  return (
    <div className="flex h-full min-w-0 flex-col">
      {/* 3xl until the pane is wide enough that 2/3 exceeds it. */}
      <div className="mx-auto flex h-full w-full max-w-[max(48rem,calc(100%*2/3))] min-w-0 flex-col">
        <div className="min-h-0 min-w-0 flex-1 overflow-hidden">
          {isEmpty ? (
            <WorkspaceEmptyState
              noAgent={props.noAgent}
              onCreateAgent={props.onCreateAgent}
            />
          ) : (
            <WorkspaceMessageList
              messages={props.messages}
              isStreaming={props.isStreaming}
              error={props.error}
              selectedToolCallId={props.selectedToolCallId}
              onSelectToolCall={props.onSelectToolCall}
              onApproveTool={props.onApproveTool}
              onDenyTool={props.onDenyTool}
              search={props.search}
              containerRef={containerRef}
              onScroll={onScroll}
            />
          )}
        </div>
        <ApprovalBar
          messages={props.messages}
          onApprove={props.onApproveTool}
          onDeny={props.onDenyTool}
        />
        <FileChangeSummary messages={props.messages} />
        <InputBar
          onSend={props.onSendMessage}
          onStop={props.onStop}
          onOptimize={optimize.onOptimize}
          disabled={props.noAgent ?? false}
          isStreaming={props.isStreaming}
          visionEnabled={props.visionEnabled}
          optimizing={optimize.optimizing}
          accessory={props.inputAccessory}
          tokenUsage={props.tokenUsage}
        />
      </div>
    </div>
  )
}
