import type { ReactNode } from "react"
import { cn } from "@/lib/utils"

export function KanbanSheet({
  children,
  className,
}: {
  children: ReactNode
  className?: string
}) {
  return (
    <div
      className={cn(
        "glass-tile rounded-2xl border border-[color:var(--glass-edge)] px-6 py-7 sm:px-9 sm:py-8",
        className
      )}
    >
      {children}
    </div>
  )
}

export function KanbanEyebrow({ children }: { children: ReactNode }) {
  return (
    <p className="text-[11px] tracking-wide text-muted-foreground">
      {children}
    </p>
  )
}
