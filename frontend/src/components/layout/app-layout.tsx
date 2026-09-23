import { BackgroundLayer } from "@/components/background/background-layer"
import { TopBar } from "./top-bar"
import { Sidebar } from "./sidebar"
import type { AgentInfo, SessionMeta, StreamStatus } from "@/types/agent"

interface AppLayoutProps {
  sidebarCollapsed: boolean
  onToggleSidebar: () => void
  onOpenSettings: () => void
  onOpenManage?: () => void
  onOpenKanban?: () => void
  onOpenApiDocs?: () => void
  onOpenUsage?: () => void
  onOpenRequestLogs?: () => void
  onOpenAdmin?: () => void
  onNewSession?: () => void
  sessions?: SessionMeta[]
  activeSessionId?: string | null
  onResumeSession?: (id: string) => void
  onDeleteSession?: (id: string) => void
  agent: AgentInfo
  agents?: AgentInfo[]
  agentLocked?: boolean
  onSelectAgent?: (id: string) => void
  status: StreamStatus
  theme: string
  onToggleTheme: () => void
  leftPanel: React.ReactNode
  /** actions rendered in the TopBar right-side button group */
  topBarActions?: React.ReactNode
}

export function AppLayout({
  leftPanel,
  topBarActions,
  agent,
  agents,
  agentLocked,
  onSelectAgent,
  status,
  theme,
  onToggleTheme,
  ...sidebarProps
}: AppLayoutProps) {
  return (
    <div className="flex h-svh overflow-hidden">
      <BackgroundLayer />
      <Sidebar
        {...sidebarProps}
        agents={agents}
        collapsed={sidebarProps.sidebarCollapsed}
        onToggle={sidebarProps.onToggleSidebar}
      />

      <div className="flex min-w-0 flex-1 flex-col">
        <TopBar
          agent={agent}
          agents={agents}
          agentLocked={agentLocked}
          onSelectAgent={onSelectAgent}
          status={status}
          theme={theme}
          onToggleTheme={onToggleTheme}
          actions={topBarActions}
        />
        <div className="flex-1 overflow-hidden">{leftPanel}</div>
      </div>
    </div>
  )
}
