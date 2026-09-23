import type { ReactNode } from "react"
import { cn } from "@/lib/utils"

export function SettingsSection({
  title,
  description,
  icon,
  footer,
  padded = true,
  children,
}: {
  title: string
  description?: string
  icon?: ReactNode
  footer?: ReactNode
  padded?: boolean
  children: ReactNode
}) {
  return (
    <section className="glass-tile overflow-hidden rounded-2xl border border-[color:var(--glass-edge)]">
      <SettingsSectionHead
        title={title}
        description={description}
        icon={icon}
      />
      <div className={cn(padded ? "px-5 pb-5" : "border-t border-border")}>
        {children}
      </div>
      {footer ? (
        <div className="flex items-center justify-end gap-2 border-t border-border px-5 py-3">
          {footer}
        </div>
      ) : null}
    </section>
  )
}

function SettingsSectionHead({
  title,
  description,
  icon,
}: {
  title: string
  description?: string
  icon?: ReactNode
}) {
  return (
    <header className="flex items-start gap-3 px-5 py-4">
      {icon ? (
        <span className="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground">
          {icon}
        </span>
      ) : null}
      <div className="min-w-0 pt-0.5">
        <h2 className="text-sm font-medium tracking-tight text-foreground">
          {title}
        </h2>
        {description ? (
          <p className="mt-0.5 text-[11px] leading-relaxed text-muted-foreground">
            {description}
          </p>
        ) : null}
      </div>
    </header>
  )
}
