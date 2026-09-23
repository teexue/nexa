import { useTranslation } from "react-i18next"
import { Moon, Sun } from "lucide-react"
import { Button } from "@/components/ui/button"
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import { StatusIndicator } from "@/components/shared/status-indicator"
import { HealthIndicator } from "@/components/monitoring/health-indicator"
import { AgentSwitcher } from "./agent-switcher"
import type { AgentInfo, StreamStatus } from "@/types/agent"

interface TopBarProps {
  agent: AgentInfo
  agents?: AgentInfo[]
  agentLocked?: boolean
  onSelectAgent?: (id: string) => void
  status: StreamStatus
  theme: string
  onToggleTheme: () => void
  /** actions rendered in the right-side button group (e.g. conversation search/export) */
  actions?: React.ReactNode
}

function TopBarButton({
  tooltip,
  onClick,
  children,
}: {
  tooltip: string
  onClick: () => void
  children: React.ReactNode
}) {
  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <Button
            variant="ghost"
            size="icon-xs"
            onClick={onClick}
            className="h-7 w-7 rounded-lg"
          />
        }
      >
        {children}
      </TooltipTrigger>
      <TooltipContent>{tooltip}</TooltipContent>
    </Tooltip>
  )
}

export function TopBar({
  agent,
  agents = [],
  agentLocked = false,
  onSelectAgent,
  status,
  theme,
  onToggleTheme,
  actions,
}: TopBarProps) {
  const { t } = useTranslation()
  return (
    <header className="glass-panel flex h-10 shrink-0 items-center justify-between border-b border-[color:var(--glass-edge)] px-4">
      <div className="flex items-center gap-2.5">
        <AgentSwitcher
          agent={agent}
          agents={agents}
          locked={agentLocked}
          onSelectAgent={onSelectAgent}
        />
        <StatusIndicator status={status} />
      </div>
      <div className="flex items-center gap-0.5">
        <div id="shell-topbar-actions" className="contents" />
        {actions}
        <HealthIndicator />
        <TopBarButton tooltip={t("layout.toggleTheme")} onClick={onToggleTheme}>
          {theme === "dark" ? (
            <Sun className="h-3.5 w-3.5 text-muted-foreground" />
          ) : (
            <Moon className="h-3.5 w-3.5 text-muted-foreground" />
          )}
        </TopBarButton>
      </div>
    </header>
  )
}
