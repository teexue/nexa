import { type LucideIcon } from "lucide-react"

interface PageHeaderProps {
  icon?: LucideIcon
  title: string
  description?: string
  actions?: React.ReactNode
}

/** Standard page header: icon + title, right-side actions. Page-level
 * navigation relies on the sidebar/router, so no back button is rendered. */
export function PageHeader({
  icon: Icon,
  title,
  description,
  actions,
}: PageHeaderProps) {
  return (
    <header className="flex items-center gap-3 border-b border-[color:var(--glass-edge)] px-6 py-4">
      <div className="flex items-center gap-2">
        {Icon && <Icon className="h-4 w-4 text-primary" />}
        <div>
          <h1 className="font-heading text-base tracking-tight text-foreground">
            {title}
          </h1>
          {description && (
            <p className="mt-0.5 text-[11px] text-muted-foreground">
              {description}
            </p>
          )}
        </div>
      </div>
      {actions && (
        <>
          <div className="flex-1" />
          <div className="flex items-center gap-2">{actions}</div>
        </>
      )}
    </header>
  )
}
