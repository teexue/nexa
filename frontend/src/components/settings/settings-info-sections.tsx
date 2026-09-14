import { useEffect, useState } from "react"
import { useTranslation } from "react-i18next"
import { ExternalLink, ImageIcon, Keyboard } from "lucide-react"
import { fetchVersion } from "@/lib/api"
import { BackgroundPanel } from "./background-panel"
import { SettingsSection } from "./settings-section"

export function BackgroundSection() {
  const { t } = useTranslation()
  return (
    <SettingsSection
      title={t("settings.background")}
      description={t("settings.backgroundHint")}
      icon={<ImageIcon className="h-3.5 w-3.5" />}
    >
      <BackgroundPanel />
    </SettingsSection>
  )
}

export function AppSection() {
  const { t } = useTranslation()
  return (
    <SettingsSection
      title={t("settings.app")}
      icon={<Keyboard className="h-3.5 w-3.5" />}
      padded={false}
    >
      <ShortcutsList />
      <AboutRow />
    </SettingsSection>
  )
}

function ShortcutsList() {
  const { t } = useTranslation()
  const shortcuts = [
    [t("settings.shortcutSidebar"), "⌘ Shift S"],
    [t("settings.shortcutClose"), "Esc"],
    [t("settings.shortcutSend"), "Enter"],
    [t("settings.shortcutNewline"), "Shift Enter"],
  ] as const
  return (
    <div className="divide-y divide-border">
      {shortcuts.map(([label, key]) => (
        <div
          key={label}
          className="flex items-center justify-between gap-4 px-5 py-3 transition-colors hover:bg-muted/30"
        >
          <span className="text-sm text-foreground">{label}</span>
          <kbd className="rounded-md border border-border bg-muted px-1.5 py-0.5 font-mono text-[10px] text-muted-foreground">
            {key}
          </kbd>
        </div>
      ))}
    </div>
  )
}

function AboutRow() {
  const { t } = useTranslation()
  const [appVersion, setAppVersion] = useState("")
  useEffect(() => {
    let cancelled = false
    fetchVersion().then((v) => {
      if (!cancelled) setAppVersion(v)
    })
    return () => {
      cancelled = true
    }
  }, [])
  return (
    <a
      href="https://github.com/teexue/nexa"
      target="_blank"
      rel="noopener noreferrer"
      className="flex items-center justify-between gap-3 border-t border-border px-5 py-3.5 transition-colors hover:bg-muted/40"
    >
      <span className="min-w-0">
        <span className="block font-mono text-xs font-medium text-foreground">
          Nexa {appVersion || "dev"}
        </span>
        <span className="mt-0.5 block text-[11px] text-muted-foreground">
          {t("settings.aboutDesc")}
        </span>
      </span>
      <ExternalLink className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
    </a>
  )
}
