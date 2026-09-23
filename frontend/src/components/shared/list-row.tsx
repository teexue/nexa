import { cn } from "@/lib/utils"

interface ListRowProps {
  onClick?: () => void
  children: React.ReactNode
  className?: string
}

/** Standard list row card with the shared hover treatment. */
export function ListRow({ onClick, children, className }: ListRowProps) {
  return (
    <div
      onClick={onClick}
      className={cn(
        "glass-tile rounded-2xl border border-[color:var(--glass-edge)] px-4 py-3 transition-colors hover:border-primary/45 hover:bg-primary/12",
        onClick && "cursor-pointer",
        className
      )}
    >
      {children}
    </div>
  )
}
