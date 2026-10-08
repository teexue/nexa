import { useState, type ReactNode } from "react"
import { Link } from "react-router"
import { useTranslation } from "react-i18next"
import {
  AlertTriangle,
  Check,
  ChevronDown,
  ChevronRight,
  Clock,
  GitBranch,
  Loader2,
  ShieldQuestion,
  X,
} from "lucide-react"
import { Button } from "@/components/ui/button"
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible"
import { InlineToolDetail } from "./inline-tool-detail"
import { businessFailed, extractInputSummary } from "./tool-summary"
import { cn } from "@/lib/utils"
import { toolDisplayName } from "@/lib/tool-i18n"
import type { ToolCallEntry } from "@/types/agent"
import type { TFunction } from "i18next"

interface ToolOperationCardProps {
  toolCall: ToolCallEntry
  isSelected: boolean
  onSelect: () => void
  onApprove?: (approvalId: string) => void
  onDeny?: (approvalId: string) => void
}

type StatusCfg = { icon: typeof Clock; color: string; label: string }

function runningStatus(t: TFunction, sub: boolean): StatusCfg {
  return sub
    ? { icon: GitBranch, color: "text-chart-2", label: t("status.delegating") }
    : {
        icon: Loader2,
        color: "text-primary",
        label: t("conversation.groupRunning"),
      }
}

function queuedStatus(t: TFunction): StatusCfg {
  return { icon: Clock, color: "text-chart-2", label: t("status.queued") }
}

function resolveStatus(toolCall: ToolCallEntry, t: TFunction): StatusCfg {
  switch (toolCall.status) {
    case "running":
      return runningStatus(t, false)
    case "sub_agent_queued":
      return queuedStatus(t)
    case "sub_agent_running":
      return runningStatus(t, true)
    case "pending_approval":
      return {
        icon: ShieldQuestion,
        color: "text-warning",
        label: t("status.pendingApproval"),
      }
    case "denied":
      return {
        icon: AlertTriangle,
        color: "text-warning",
        label: t("status.denied"),
      }
    case "error":
      return { icon: X, color: "text-destructive", label: t("status.failed") }
    case "completed":
      return businessFailed(toolCall)
        ? { icon: X, color: "text-destructive", label: t("status.failed") }
        : { icon: Check, color: "text-success", label: t("status.success") }
    default:
      return {
        icon: Clock,
        color: "text-muted-foreground",
        label: t("status.pending"),
      }
  }
}

function StatusIcon({ status, config }: { status: string; config: StatusCfg }) {
  if (
    status === "running" ||
    status === "sub_agent_queued" ||
    status === "sub_agent_running"
  )
    return (
      <Loader2 className={cn("h-3 w-3 shrink-0 animate-spin", config.color)} />
    )
  return <config.icon className={cn("h-3 w-3 shrink-0", config.color)} />
}

export function ToolOperationCard({
  toolCall,
  isSelected,
}: ToolOperationCardProps) {
  const { t } = useTranslation()
  const [expanded, setExpanded] = useState(false)
  const config = resolveStatus(toolCall, t)
  return (
    <Collapsible open={expanded} onOpenChange={setExpanded}>
      <ToolCardTrigger
        toolCall={toolCall}
        isSelected={isSelected}
        expanded={expanded}
        config={config}
        inputSummary={extractInputSummary(toolCall.name, toolCall.input)}
      />
      <CollapsibleContent>
        <div className="py-1 pl-5">
          <InlineToolDetail toolCall={toolCall} />
          <SubAgentDetailLink
            sessionId={
              toolCall.sessionId || sessionIdFromOutput(toolCall.output)
            }
          />
          {toolCall.status === "denied" && (
            <p className="mt-1.5 text-xs text-warning">
              {t("conversation.toolDenied")}
            </p>
          )}
        </div>
      </CollapsibleContent>
    </Collapsible>
  )
}

function ToolCardTrigger({
  toolCall,
  isSelected,
  expanded,
  config,
  inputSummary,
}: {
  toolCall: ToolCallEntry
  isSelected: boolean
  expanded: boolean
  config: StatusCfg
  inputSummary: ReactNode
}) {
  const { t } = useTranslation()
  return (
    <CollapsibleTrigger
      render={
        <Button
          variant="ghost"
          size="sm"
          className={cn(
            "h-auto w-full justify-start gap-1.5 rounded-lg px-2 py-1 text-left font-mono text-[11px]",
            isSelected
              ? "text-foreground"
              : "text-muted-foreground hover:text-foreground"
          )}
        />
      }
    >
      <StatusIcon status={toolCall.status} config={config} />
      {expanded ? (
        <ChevronDown className="h-3 w-3 shrink-0" />
      ) : (
        <ChevronRight className="h-3 w-3 shrink-0" />
      )}
      <span className="shrink-0 text-foreground">
        {toolDisplayName(toolCall.name, t)}
      </span>
      {inputSummary && (
        <span className="min-w-0 flex-1 truncate">{inputSummary}</span>
      )}
    </CollapsibleTrigger>
  )
}

function sessionIdFromOutput(output: unknown): string | undefined {
  if (!output || typeof output !== "object" || Array.isArray(output)) return
  const sid = (output as Record<string, unknown>).session_id
  return typeof sid === "string" && sid ? sid : undefined
}

function SubAgentDetailLink({ sessionId }: { sessionId?: string }) {
  const { t } = useTranslation()
  if (!sessionId) return null
  return (
    <Link
      to={`/sessions/${encodeURIComponent(sessionId)}`}
      className="mt-1.5 inline-block text-[11px] text-primary hover:underline"
    >
      {t("conversation.viewSubAgentDetail")}
    </Link>
  )
}
