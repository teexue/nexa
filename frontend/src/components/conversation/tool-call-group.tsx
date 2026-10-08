import { ToolOperationCard } from "./tool-operation-card"
import { SubAgentCard } from "./sub-agent-card"
import { isSubAgentCall } from "@/lib/sub-agent"
import type { ToolCallEntry } from "@/types/agent"

interface ToolCallGroupProps {
  toolCalls: ToolCallEntry[]
  foldKey: string
  selectedToolCallId: string | null
  onSelectToolCall: (id: string) => void
  onApproveTool?: (approvalId: string) => void
  onDenyTool?: (approvalId: string) => void
}

export function ToolCallGroup(props: ToolCallGroupProps) {
  return (
    <div className="flex flex-col gap-0.5">
      {props.toolCalls.map((tc) =>
        isSubAgentCall(tc) ? (
          <SubAgentCard key={tc.id} toolCall={tc} />
        ) : (
          <ToolOperationCard
            key={`${tc.id}:${props.foldKey}`}
            toolCall={tc}
            isSelected={props.selectedToolCallId === tc.id}
            onSelect={() => props.onSelectToolCall(tc.id)}
            onApprove={props.onApproveTool}
            onDeny={props.onDenyTool}
          />
        )
      )}
    </div>
  )
}
