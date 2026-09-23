import { cn } from "@/lib/utils"

interface PageShellProps {
  children: React.ReactNode
  className?: string
}

/** Standard page root: full-height column with background. */
export function PageShell({ children, className }: PageShellProps) {
  return <div className={cn("flex h-full flex-col", className)}>{children}</div>
}

interface PageMainProps {
  children: React.ReactNode
  className?: string
  contentClassName?: string
}

/** Standard scrollable content area with the shared page padding. */
export function PageMain({
  children,
  className,
  contentClassName,
}: PageMainProps) {
  return (
    <main className={cn("min-h-0 flex-1 overflow-auto", className)}>
      <div className={cn("px-6 py-6", contentClassName)}>{children}</div>
    </main>
  )
}
