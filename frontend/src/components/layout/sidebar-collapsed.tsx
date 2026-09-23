import { useTranslation } from "react-i18next"
import {
  BookOpen,
  ChevronRight,
  Coins,
  KanbanSquare,
  Layers,
  Plus,
  ScrollText,
  Settings,
  ShieldCheck,
} from "lucide-react"
import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import { useAuth } from "@/lib/auth"

export interface CollapsedSidebarProps {
  onToggle: () => void
  onOpenSettings: () => void
  onOpenManage?: () => void
  onOpenKanban?: () => void
  onOpenApiDocs?: () => void
  onOpenUsage?: () => void
  onOpenRequestLogs?: () => void
  onOpenAdmin?: () => void
  onNewSession?: () => void
}

function CollapsedIconButton({
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
            className="rounded-lg"
          />
        }
      >
        {children}
      </TooltipTrigger>
      <TooltipContent side="right">{tooltip}</TooltipContent>
    </Tooltip>
  )
}

function CollapsedWorkspaceIcons({
  onOpenKanban,
  onOpenApiDocs,
  onOpenUsage,
  onOpenRequestLogs,
}: Pick<
  CollapsedSidebarProps,
  "onOpenKanban" | "onOpenApiDocs" | "onOpenUsage" | "onOpenRequestLogs"
>) {
  const { t } = useTranslation()
  const isAdmin = useAuth().user?.role === "admin"
  return (
    <>
      {onOpenKanban && (
        <CollapsedIconButton
          tooltip={t("layout.kanban")}
          onClick={onOpenKanban}
        >
          <KanbanSquare className="h-3.5 w-3.5" />
        </CollapsedIconButton>
      )}
      {onOpenApiDocs && (
        <CollapsedIconButton
          tooltip={t("layout.apiDocs")}
          onClick={onOpenApiDocs}
        >
          <BookOpen className="h-3.5 w-3.5" />
        </CollapsedIconButton>
      )}
      {onOpenUsage && (
        <CollapsedIconButton tooltip={t("layout.usage")} onClick={onOpenUsage}>
          <Coins className="h-3.5 w-3.5" />
        </CollapsedIconButton>
      )}
      {isAdmin && onOpenRequestLogs && (
        <CollapsedIconButton
          tooltip={t("layout.requestLogs")}
          onClick={onOpenRequestLogs}
        >
          <ScrollText className="h-3.5 w-3.5" />
        </CollapsedIconButton>
      )}
    </>
  )
}

function CollapsedNavItems({
  onOpenSettings,
  onOpenManage,
  onOpenKanban,
  onOpenApiDocs,
  onOpenUsage,
  onOpenRequestLogs,
  onOpenAdmin,
}: Omit<CollapsedSidebarProps, "onToggle" | "onNewSession">) {
  const { t } = useTranslation()
  const isAdmin = useAuth().user?.role === "admin"
  return (
    <>
      {onOpenManage && (
        <CollapsedIconButton
          tooltip={t("layout.manage")}
          onClick={onOpenManage}
        >
          <Layers className="h-3.5 w-3.5" />
        </CollapsedIconButton>
      )}
      <CollapsedWorkspaceIcons
        onOpenKanban={onOpenKanban}
        onOpenApiDocs={onOpenApiDocs}
        onOpenUsage={onOpenUsage}
        onOpenRequestLogs={onOpenRequestLogs}
      />
      {isAdmin && onOpenAdmin && (
        <CollapsedIconButton tooltip={t("layout.admin")} onClick={onOpenAdmin}>
          <ShieldCheck className="h-3.5 w-3.5" />
        </CollapsedIconButton>
      )}
      <CollapsedIconButton
        tooltip={t("common.settings")}
        onClick={onOpenSettings}
      >
        <Settings className="h-3.5 w-3.5" />
      </CollapsedIconButton>
    </>
  )
}

export function CollapsedSidebar({
  onToggle,
  onOpenSettings,
  onOpenManage,
  onOpenKanban,
  onOpenApiDocs,
  onOpenUsage,
  onOpenRequestLogs,
  onOpenAdmin,
  onNewSession,
}: CollapsedSidebarProps) {
  const { t } = useTranslation()
  return (
    <div className="glass-panel flex h-full w-12 flex-col items-center gap-1 border-r border-[color:var(--glass-edge)] py-3">
      <CollapsedIconButton
        tooltip={t("layout.expandSidebar")}
        onClick={onToggle}
      >
        <ChevronRight className="h-3.5 w-3.5" />
      </CollapsedIconButton>
      <Separator className="my-2 w-6" />
      {onNewSession && (
        <CollapsedIconButton
          tooltip={t("layout.newSession")}
          onClick={onNewSession}
        >
          <Plus className="h-3.5 w-3.5" />
        </CollapsedIconButton>
      )}
      <div className="flex-1" />
      <CollapsedNavItems
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
