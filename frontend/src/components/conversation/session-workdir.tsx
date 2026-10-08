import { useState } from "react"
import { useTranslation } from "react-i18next"
import { Check, Folder, FolderOpen, FolderPlus, X } from "lucide-react"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { DirPickerDialog } from "@/components/settings/dir-picker-dialog"
import { cn } from "@/lib/utils"
import {
  basename,
  clearWorkdirHistory,
  loadWorkdirHistory,
  pushWorkdirHistory,
  removeWorkdirHistory,
} from "@/components/settings/dir-picker-path"

interface SessionWorkdirProps {
  workDir: string
  sessionScoped: boolean
  onPick: (dir: string) => void
  onClear: () => void
}

const chipBtn =
  "h-6 min-w-0 max-w-full shrink justify-start gap-1 overflow-hidden rounded-md px-1.5 text-muted-foreground hover:text-foreground"

/** Per-session working directory: recents menu, then file-manager picker. */
export function SessionWorkdir({
  workDir,
  sessionScoped,
  onPick,
  onClear,
}: SessionWorkdirProps) {
  const [pickerOpen, setPickerOpen] = useState(false)
  const [history, setHistory] = useState<string[]>(loadWorkdirHistory)
  const handlePick = (dir: string) => {
    setHistory(pushWorkdirHistory(dir))
    onPick(dir)
  }
  return (
    <div className="max-w-full min-w-0 overflow-hidden">
      <WorkdirMenu
        workDir={workDir}
        history={history}
        sessionScoped={sessionScoped}
        onRefresh={() => setHistory(loadWorkdirHistory())}
        onPick={handlePick}
        onRemove={(dir) => setHistory(removeWorkdirHistory(dir))}
        onClearRecents={() => setHistory(clearWorkdirHistory())}
        onBrowse={() => setPickerOpen(true)}
        onClear={onClear}
      />
      <DirPickerDialog
        open={pickerOpen}
        onOpenChange={setPickerOpen}
        initialPath={workDir}
        recents={history}
        onSelect={handlePick}
        onReset={sessionScoped ? onClear : undefined}
      />
    </div>
  )
}

function WorkdirMenu({
  workDir,
  history,
  sessionScoped,
  onRefresh,
  onPick,
  onRemove,
  onClearRecents,
  onBrowse,
  onClear,
}: {
  workDir: string
  history: string[]
  sessionScoped: boolean
  onRefresh: () => void
  onPick: (dir: string) => void
  onRemove: (dir: string) => void
  onClearRecents: () => void
  onBrowse: () => void
  onClear: () => void
}) {
  const { t } = useTranslation()
  return (
    <DropdownMenu
      onOpenChange={(open) => {
        if (open) onRefresh()
      }}
    >
      <DropdownMenuTrigger
        render={
          <Button
            type="button"
            variant="ghost"
            size="sm"
            className={chipBtn}
            title={workDir || t("conversation.workdirDefaultHint")}
          />
        }
      >
        <FolderOpen className="h-3.5 w-3.5 shrink-0" />
        <span className="min-w-0 truncate font-mono text-[11px]">
          {workDir ? basename(workDir) : t("conversation.workdirDefault")}
        </span>
      </DropdownMenuTrigger>
      <WorkdirMenuContent
        history={history}
        workDir={workDir}
        sessionScoped={sessionScoped}
        onPick={onPick}
        onRemove={onRemove}
        onClearRecents={onClearRecents}
        onBrowse={onBrowse}
        onClear={onClear}
      />
    </DropdownMenu>
  )
}

function WorkdirMenuContent({
  history,
  workDir,
  sessionScoped,
  onPick,
  onRemove,
  onClearRecents,
  onBrowse,
  onClear,
}: {
  history: string[]
  workDir: string
  sessionScoped: boolean
  onPick: (dir: string) => void
  onRemove: (dir: string) => void
  onClearRecents: () => void
  onBrowse: () => void
  onClear: () => void
}) {
  return (
    <DropdownMenuContent
      side="top"
      align="start"
      className="menu-opaque w-72 min-w-72 rounded-xl p-1.5"
    >
      <WorkdirRecents
        history={history}
        workDir={workDir}
        onPick={onPick}
        onRemove={onRemove}
        onClearRecents={onClearRecents}
      />
      <WorkdirMenuActions
        sessionScoped={sessionScoped}
        onBrowse={onBrowse}
        onClear={onClear}
      />
    </DropdownMenuContent>
  )
}

function WorkdirRecents({
  history,
  workDir,
  onPick,
  onRemove,
  onClearRecents,
}: {
  history: string[]
  workDir: string
  onPick: (dir: string) => void
  onRemove: (dir: string) => void
  onClearRecents: () => void
}) {
  const { t } = useTranslation()
  if (history.length === 0) {
    return (
      <p className="px-2.5 pt-2 pb-1 text-[11px] leading-4 text-muted-foreground">
        {t("conversation.workdirDefaultHint")}
      </p>
    )
  }
  return (
    <>
      <WorkdirRecentsHead onClear={onClearRecents} />
      {history.map((dir) => (
        <WorkdirRecentRow
          key={dir}
          dir={dir}
          selected={dir === workDir}
          onPick={onPick}
          onRemove={onRemove}
        />
      ))}
      <DropdownMenuSeparator />
    </>
  )
}

function WorkdirRecentsHead({ onClear }: { onClear: () => void }) {
  const { t } = useTranslation()
  return (
    <div className="flex items-center justify-between gap-2 px-2 pt-1 pb-0.5">
      <span className="text-[11px] text-muted-foreground">
        {t("settings.dirRecents")}
      </span>
      <DropdownMenuItem
        onClick={onClear}
        className="h-auto rounded-md px-1.5 py-0.5 text-[11px] font-normal text-muted-foreground"
      >
        {t("conversation.workdirClearRecents")}
      </DropdownMenuItem>
    </div>
  )
}

function WorkdirMenuActions({
  sessionScoped,
  onBrowse,
  onClear,
}: {
  sessionScoped: boolean
  onBrowse: () => void
  onClear: () => void
}) {
  const { t } = useTranslation()
  return (
    <>
      <DropdownMenuItem
        onClick={onBrowse}
        className="gap-2 rounded-lg px-2 py-1.5 text-xs font-normal"
      >
        <WorkdirMark tone="add" />
        {t("conversation.workdirAdd")}
      </DropdownMenuItem>
      {sessionScoped ? (
        <DropdownMenuItem
          onClick={onClear}
          className="gap-2 rounded-lg px-2 py-1.5 text-xs font-normal text-muted-foreground"
        >
          <span className="flex h-6 w-6 shrink-0 items-center justify-center">
            <FolderOpen className="h-3.5 w-3.5" />
          </span>
          {t("conversation.workdirReset")}
        </DropdownMenuItem>
      ) : null}
    </>
  )
}

function WorkdirRecentRow({
  dir,
  selected,
  onPick,
  onRemove,
}: {
  dir: string
  selected: boolean
  onPick: (dir: string) => void
  onRemove: (dir: string) => void
}) {
  const { t } = useTranslation()
  return (
    <DropdownMenuItem
      onClick={() => onPick(dir)}
      className={cn(
        "items-start gap-2 rounded-lg px-2 py-1.5 font-normal",
        selected && "bg-primary/12"
      )}
    >
      <WorkdirMark />
      <WorkdirPath name={basename(dir)} parent={parentPath(dir)} title={dir} />
      <WorkdirRowTools
        selected={selected}
        forgetLabel={t("conversation.workdirForget")}
        onRemove={() => onRemove(dir)}
      />
    </DropdownMenuItem>
  )
}

function WorkdirPath({
  name,
  parent,
  title,
}: {
  name: string
  parent: string
  title: string
}) {
  return (
    <span className="min-w-0 flex-1 py-0.5" title={title}>
      <span className="block truncate text-xs">{name}</span>
      {parent ? (
        <span className="block truncate font-mono text-[10px] leading-4 text-muted-foreground">
          {parent}
        </span>
      ) : null}
    </span>
  )
}

function WorkdirRowTools({
  selected,
  forgetLabel,
  onRemove,
}: {
  selected: boolean
  forgetLabel: string
  onRemove: () => void
}) {
  return (
    <span className="flex shrink-0 items-center gap-0.5 pt-0.5">
      {selected ? (
        <Check className="h-3.5 w-3.5 text-primary" />
      ) : (
        <span className="h-3.5 w-3.5" />
      )}
      <ForgetButton label={forgetLabel} onRemove={onRemove} />
    </span>
  )
}

function WorkdirMark({ tone }: { tone?: "add" }) {
  const add = tone === "add"
  return (
    <span
      className={cn(
        "flex h-6 w-6 shrink-0 items-center justify-center rounded-md",
        add ? "bg-primary/12 text-primary" : "bg-muted text-muted-foreground"
      )}
    >
      {add ? (
        <FolderPlus className="h-3.5 w-3.5" />
      ) : (
        <Folder className="h-3.5 w-3.5" />
      )}
    </span>
  )
}

function ForgetButton({
  label,
  onRemove,
}: {
  label: string
  onRemove: () => void
}) {
  return (
    <span
      role="button"
      tabIndex={-1}
      title={label}
      className="flex h-5 w-5 items-center justify-center rounded-md text-muted-foreground opacity-0 group-hover/dropdown-menu-item:opacity-100 group-focus/dropdown-menu-item:opacity-100 hover:bg-muted hover:text-foreground"
      onClick={(e) => {
        e.preventDefault()
        e.stopPropagation()
        onRemove()
      }}
      onPointerDown={(e) => {
        e.preventDefault()
        e.stopPropagation()
      }}
    >
      <X className="h-3 w-3" />
    </span>
  )
}

function parentPath(path: string): string {
  const trimmed = path.replace(/[\\/]+$/, "")
  const slash = Math.max(trimmed.lastIndexOf("/"), trimmed.lastIndexOf("\\"))
  if (slash <= 0) return ""
  return trimmed.slice(0, slash)
}
