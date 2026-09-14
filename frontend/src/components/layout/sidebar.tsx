import { useTranslation } from "react-i18next"
import { ChevronLeft, Plus } from "lucide-react"
import { Button } from "@/components/ui/button"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Separator } from "@/components/ui/separator"
import type { AgentInfo, SessionMeta } from "@/types/agent"
import { SessionList } from "./sidebar-session-list"
import { CollapsedSidebar } from "./sidebar-collapsed"
import { SidebarFooter } from "./sidebar-footer"

interface SidebarProps {
  collapsed: boolean
  onToggle: () => void
  onOpenSettings: () => void
  onOpenManage?: () => void
  onOpenKanban?: () => void
  onOpenApiDocs?: () => void
  onOpenUsage?: () => void
  onOpenRequestLogs?: () => void
  onOpenAdmin?: () => void
  onNewSession?: () => void
  sessions?: SessionMeta[]
  agents?: AgentInfo[]
  activeSessionId?: string | null
  onResumeSession?: (id: string) => void
  onDeleteSession?: (id: string) => void
}

function buildAgentLabels(agents: AgentInfo[]): Record<string, string> {
  const labels: Record<string, string> = {}
  for (const a of agents) {
    if (a.id) labels[a.id] = a.name
    labels[a.name] = a.name
  }
  return labels
}

function SidebarBrand({ onToggle }: { onToggle: () => void }) {
  return (
    <div className="flex items-center justify-between px-3.5 py-3">
      <div className="flex items-center gap-2.5">
        <img src="/logo.png" alt="Nexa logo" className="h-7 w-7 rounded-lg" />
        <span className="text-sm font-medium tracking-tight text-foreground">
          Nexa
        </span>
      </div>
      <Button
        variant="ghost"
        size="icon-xs"
        onClick={onToggle}
        className="h-6 w-6 rounded-lg text-muted-foreground"
      >
        <ChevronLeft className="h-3.5 w-3.5" />
      </Button>
    </div>
  )
}

function NewSessionButton({ onClick }: { onClick?: () => void }) {
  const { t } = useTranslation()
  return (
    <div className="p-2.5">
      <Button
        variant="outline"
        size="sm"
        className="w-full justify-start gap-2 rounded-xl text-xs text-muted-foreground"
        onClick={onClick}
      >
        <Plus className="h-3.5 w-3.5" /> {t("layout.newSession")}
      </Button>
    </div>
  )
}

export function Sidebar(props: SidebarProps) {
  if (props.collapsed) {
    return (
      <CollapsedSidebar
        onToggle={props.onToggle}
        onOpenSettings={props.onOpenSettings}
        onOpenManage={props.onOpenManage}
        onOpenKanban={props.onOpenKanban}
        onOpenApiDocs={props.onOpenApiDocs}
        onOpenUsage={props.onOpenUsage}
        onOpenRequestLogs={props.onOpenRequestLogs}
        onOpenAdmin={props.onOpenAdmin}
        onNewSession={props.onNewSession}
      />
    )
  }
  return <ExpandedSidebar {...props} />
}

function ExpandedSidebar({
  onToggle,
  onOpenSettings,
  onOpenManage,
  onOpenKanban,
  onOpenApiDocs,
  onOpenUsage,
  onOpenRequestLogs,
  onOpenAdmin,
  onNewSession,
  sessions = [],
  agents = [],
  activeSessionId,
  onResumeSession,
  onDeleteSession,
}: SidebarProps) {
  return (
    <div className="flex h-full w-60 shrink-0 flex-col overflow-hidden border-r border-border bg-sidebar">
      <SidebarBrand onToggle={onToggle} />
      <Separator />
      <NewSessionButton onClick={onNewSession} />
      <ScrollArea className="min-h-0 flex-1">
        <SessionList
          sessions={sessions}
          activeSessionId={activeSessionId}
          onResumeSession={onResumeSession}
          onDeleteSession={onDeleteSession}
          agentLabels={buildAgentLabels(agents)}
        />
      </ScrollArea>
      <SidebarFooter
        onOpenSettings={onOpenSettings}
        onOpenManage={onOpenManage}
        onOpenKanban={onOpenKanban}
        onOpenApiDocs={onOpenApiDocs}
        onOpenUsage={onOpenUsage}
        onOpenRequestLogs={onOpenRequestLogs}
        onOpenAdmin={onOpenAdmin}
      />
    </div>
  )
}
