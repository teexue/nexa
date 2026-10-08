import { useTranslation } from "react-i18next"
import { Brain, Check } from "lucide-react"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import type { ProviderInfo } from "@/types/agent"
import {
  acceptedThinkEffort,
  thinkChoices,
  thinkEffortLabelKey,
  type ThinkTarget,
} from "@/lib/think-effort"

const chipBtn =
  "h-6 min-w-0 shrink justify-start gap-1 rounded-md px-1.5 text-muted-foreground hover:text-foreground"

export function ThinkEffortControl({
  providers,
  provider,
  model,
  value,
  onChange,
}: {
  providers: ProviderInfo[]
  provider: string
  model: string
  value: string
  onChange: (next: string) => void
}) {
  const info = providers.find((item) => item.name === provider)
  if (!info) return null
  return (
    <ThinkEffortPicker
      profile={{
        style: info.api_style,
        baseUrl: info.base_url,
        vendor: info.name,
        model,
      }}
      value={value}
      onChange={onChange}
    />
  )
}

function ThinkEffortPicker({
  profile,
  value,
  onChange,
}: {
  profile: ThinkTarget
  value: string
  onChange: (next: string) => void
}) {
  const choices = thinkChoices(profile)
  if (choices.length === 0) return null
  const current = acceptedThinkEffort(profile, value)
  return (
    <ThinkEffortMenu choices={choices} current={current} onChange={onChange} />
  )
}

function ThinkEffortMenu({
  choices,
  current,
  onChange,
}: {
  choices: string[]
  current: string
  onChange: (next: string) => void
}) {
  const { t } = useTranslation()
  const label = t(thinkEffortLabelKey(current))
  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <Button
            variant="ghost"
            size="sm"
            className={chipBtn}
            title={t("conversation.thinkEffort")}
          />
        }
      >
        <Brain className="h-3.5 w-3.5 shrink-0" />
        <span className="text-[11px]">{label}</span>
      </DropdownMenuTrigger>
      <DropdownMenuContent
        side="top"
        align="start"
        className="menu-opaque w-max min-w-36 rounded-xl"
      >
        <EffortItem
          id=""
          selected={current === ""}
          label={t("conversation.thinkDefault")}
          onChange={onChange}
        />
        {choices.map((id) => (
          <EffortItem
            key={id}
            id={id}
            selected={id === current}
            label={t(thinkEffortLabelKey(id))}
            onChange={onChange}
          />
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

function EffortItem({
  id,
  selected,
  label,
  onChange,
}: {
  id: string
  selected: boolean
  label: string
  onChange: (next: string) => void
}) {
  return (
    <DropdownMenuItem
      onClick={() => onChange(id)}
      className="gap-2 text-xs font-normal"
    >
      {selected ? (
        <Check className="h-3.5 w-3.5 shrink-0" />
      ) : (
        <span className="h-3.5 w-3.5 shrink-0" />
      )}
      <span>{label}</span>
    </DropdownMenuItem>
  )
}
