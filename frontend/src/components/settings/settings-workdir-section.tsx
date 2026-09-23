import { useState } from "react"
import { useTranslation } from "react-i18next"
import { FolderOpen } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { DirPickerDialog } from "./dir-picker-dialog"
import { loadWorkdirHistory, pushWorkdirHistory } from "./dir-picker-path"
import { SettingsSection } from "./settings-section"

export function WorkDirSection() {
  const { t } = useTranslation()
  const [workDir, setWorkDir] = useState(
    () => localStorage.getItem("workDir") || ""
  )
  const [pickerOpen, setPickerOpen] = useState(false)
  const handleWorkDirChange = (value: string) => {
    setWorkDir(value)
    localStorage.setItem("workDir", value)
    if (value) pushWorkdirHistory(value)
  }
  return (
    <SettingsSection
      title={t("settings.workDir")}
      description={t("settings.workDirHint")}
      icon={<FolderOpen className="h-3.5 w-3.5" />}
    >
      <WorkDirInput
        value={workDir}
        onChange={handleWorkDirChange}
        onBrowse={() => setPickerOpen(true)}
      />
      <DirPickerDialog
        open={pickerOpen}
        onOpenChange={setPickerOpen}
        initialPath={workDir}
        recents={loadWorkdirHistory()}
        onSelect={handleWorkDirChange}
      />
    </SettingsSection>
  )
}

function WorkDirInput({
  value,
  onChange,
  onBrowse,
}: {
  value: string
  onChange: (value: string) => void
  onBrowse: () => void
}) {
  const { t } = useTranslation()
  return (
    <div className="glass-tile flex overflow-hidden rounded-xl border border-[color:var(--glass-edge)] focus-within:border-primary/45">
      <Input
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={t("settings.workDirPlaceholder")}
        className="h-10 rounded-none border-0 bg-transparent font-mono text-xs shadow-none focus-visible:ring-0"
      />
      <Button
        variant="ghost"
        size="sm"
        className="h-10 shrink-0 rounded-none border-l border-border px-3 text-xs"
        onClick={onBrowse}
      >
        <FolderOpen className="h-3.5 w-3.5" /> {t("settings.browse")}
      </Button>
    </div>
  )
}
