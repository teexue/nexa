import { useTranslation } from "react-i18next"
import type { StreamStatus } from "@/types/agent"
import { cn } from "@/lib/utils"
import type { TFunction } from "i18next"

function getStatusConfig(
  t: TFunction
): Record<StreamStatus, { dot: string; label: string; className?: string }> {
  return {
    idle: {
      dot: "bg-muted-foreground/30",
      label: t("status.idle"),
    },
    streaming: {
      dot: "status-led-run",
      label: t("status.running"),
    },
    error: {
      dot: "bg-destructive",
      label: t("status.error"),
    },
    done: {
      dot: "bg-live",
      label: t("status.done"),
    },
  }
}

export function StatusIndicator({ status }: { status: StreamStatus }) {
  const { t } = useTranslation()
  const config = getStatusConfig(t)[status]
  return (
    <div className="flex items-center gap-1.5">
      <div
        className={cn("h-1.5 w-1.5 rounded-full", config.dot, config.className)}
      />
      <span className="text-[10px] font-medium text-muted-foreground">
        {config.label}
      </span>
    </div>
  )
}
