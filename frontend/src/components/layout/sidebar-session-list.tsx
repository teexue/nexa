import { useState } from "react"
import { useTranslation } from "react-i18next"
import { Clock, Loader2, Trash2 } from "lucide-react"
import { Button } from "@/components/ui/button"
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import { ConfirmDeleteDialog } from "@/components/settings/confirm-delete-dialog"
import type { SessionMeta } from "@/types/agent"
import { formatRelativeTime } from "@/lib/format"

interface SessionListProps {
  sessions: SessionMeta[]
  activeSessionId?: string | null
  onResumeSession?: (id: string) => void
  onDeleteSession?: (id: string) => void
  agentLabels?: Record<string, string>
}

function SessionActionBtn({
  tooltip,
  onClick,
  children,
}: {
  tooltip: string
  onClick: (e: React.MouseEvent) => void
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
            className="h-5 w-5 rounded-md"
          />
        }
      >
        {children}
      </TooltipTrigger>
      <TooltipContent side="right">{tooltip}</TooltipContent>
    </Tooltip>
  )
}

function SessionListItem({
  sess,
  active,
  onResume,
  onDelete,
  agentLabel,
}: {
  sess: SessionMeta
  active: boolean
  onResume?: (id: string) => void
  onDelete?: (id: string) => void
  agentLabel?: string
}) {
  const { t } = useTranslation()
  const title = sess.title?.trim() || t("layout.untitledSession")
  return (
    <div
      className={`group flex items-center gap-2 rounded-lg px-2.5 py-2 text-left transition-all ${active ? "bg-primary/12 text-foreground" : "text-sidebar-foreground hover:bg-primary/12"}`}
    >
      <button
        onClick={() => onResume?.(sess.id)}
        className="flex min-w-0 flex-1 items-center gap-2"
      >
        <span
          className={`h-1.5 w-1.5 shrink-0 rounded-full transition-colors ${active ? "bg-primary" : "bg-muted-foreground/25 group-hover:bg-primary/40"}`}
        />
        <div className="min-w-0 flex-1">
          <span className="block truncate text-xs font-medium">{title}</span>
          <span className="block truncate text-[11px] text-muted-foreground">
            {agentLabel || sess.agent} · {formatRelativeTime(sess.updated_at)}
          </span>
        </div>
        {sess.running && (
          <Tooltip>
            <TooltipTrigger render={<span data-slot="running-indicator" />}>
              <Loader2 className="h-3 w-3 shrink-0 animate-spin text-live" />
            </TooltipTrigger>
            <TooltipContent side="right">
              {t("layout.sessionRunning")}
            </TooltipContent>
          </Tooltip>
        )}
      </button>
      <div className="flex shrink-0 items-center gap-0.5 opacity-0 transition-opacity group-hover:opacity-100">
        {onDelete && (
          <SessionActionBtn
            tooltip={t("layout.deleteSession")}
            onClick={(e) => {
              e.stopPropagation()
              onDelete(sess.id)
            }}
          >
            <Trash2 className="h-3 w-3 text-muted-foreground hover:text-destructive" />
          </SessionActionBtn>
        )}
      </div>
    </div>
  )
}

export function SessionList({
  sessions,
  activeSessionId,
  onResumeSession,
  onDeleteSession,
  agentLabels,
}: SessionListProps) {
  const { t } = useTranslation()
  const [pendingId, setPendingId] = useState<string | null>(null)
  const pending = sessions.find((s) => s.id === pendingId)
  if (sessions.length === 0) return null
  const pendingTitle = pending?.title?.trim() || t("layout.untitledSession")
  return (
    <div className="p-2.5">
      <div className="mb-2 flex items-center gap-1.5 px-2 text-[11px] font-semibold tracking-widest text-muted-foreground/70 uppercase">
        <Clock className="h-3 w-3" /> {t("layout.historySessions")}
      </div>
      <div className="flex flex-col gap-0.5">
        {sessions.map((sess) => (
          <SessionListItem
            key={sess.id}
            sess={sess}
            active={activeSessionId === sess.id}
            onResume={onResumeSession}
            onDelete={setPendingId}
            agentLabel={agentLabels?.[sess.agent]}
          />
        ))}
      </div>
      <ConfirmDeleteDialog
        open={pendingId !== null}
        title={t("layout.deleteSession")}
        message={t("layout.deleteSessionConfirm", { title: pendingTitle })}
        error={null}
        deleting={false}
        onClose={() => setPendingId(null)}
        onConfirm={() => {
          if (!pendingId) return
          const id = pendingId
          setPendingId(null)
          onDeleteSession?.(id)
        }}
      />
    </div>
  )
}
