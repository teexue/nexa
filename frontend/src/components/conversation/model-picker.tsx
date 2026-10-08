import { useTranslation } from "react-i18next"
import { Check, Cpu, Lock } from "lucide-react"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import type { ProviderInfo } from "@/types/agent"
import {
  chatModelOptions,
  modelChoiceKey,
  type ChatModelOption,
} from "@/lib/provider-models"

const chipBtn =
  "h-6 min-w-0 max-w-full shrink justify-start gap-1 overflow-hidden rounded-md px-1.5 text-muted-foreground hover:text-foreground"

export function ModelPicker({
  providers,
  provider,
  model,
  locked,
  onChange,
}: {
  providers: ProviderInfo[]
  provider: string
  model: string
  locked: boolean
  onChange: (next: { provider: string; model: string }) => void
}) {
  const { t } = useTranslation()
  const options = chatModelOptions(providers)
  const current = options.find(
    (o) => o.provider === provider && o.model === model
  )
  const title = current
    ? `${current.model} · ${current.providerLabel}`
    : t("conversation.selectModel")
  return (
    <div className="max-w-full min-w-0 overflow-hidden">
      <ModelPickerBody
        options={options}
        locked={locked}
        model={model}
        title={title}
        provider={provider}
        onChange={onChange}
      />
    </div>
  )
}

function ModelPickerBody({
  options,
  locked,
  model,
  title,
  provider,
  onChange,
}: {
  options: ChatModelOption[]
  locked: boolean
  model: string
  title: string
  provider: string
  onChange: (next: { provider: string; model: string }) => void
}) {
  const { t } = useTranslation()
  if (options.length === 0) {
    return (
      <span
        className="min-w-0 truncate px-1.5 text-[11px] text-muted-foreground"
        title={t("conversation.noEnabledModels")}
      >
        {t("conversation.noEnabledModels")}
      </span>
    )
  }
  if (locked) {
    return <ModelChip model={model} title={title} locked />
  }
  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <Button variant="ghost" size="sm" className={chipBtn} title={title} />
        }
      >
        <ModelChipInner
          model={model}
          placeholder={t("conversation.selectModel")}
        />
      </DropdownMenuTrigger>
      <ModelMenu
        options={options}
        provider={provider}
        model={model}
        onChange={onChange}
      />
    </DropdownMenu>
  )
}

function ModelChipInner({
  model,
  placeholder,
}: {
  model: string
  placeholder: string
}) {
  return (
    <>
      <Cpu className="h-3.5 w-3.5 shrink-0" />
      <span className="min-w-0 truncate font-mono text-[11px]">
        {model || placeholder}
      </span>
    </>
  )
}

function ModelChip({
  model,
  title,
  locked,
}: {
  model: string
  title: string
  locked?: boolean
}) {
  const { t } = useTranslation()
  return (
    <span
      className="inline-flex h-6 max-w-full min-w-0 items-center gap-1 overflow-hidden rounded-md px-1.5 text-muted-foreground"
      title={title}
    >
      <ModelChipInner
        model={model}
        placeholder={t("conversation.selectModel")}
      />
      {locked && <Lock className="h-3 w-3 shrink-0 opacity-70" />}
    </span>
  )
}

function ModelMenu({
  options,
  provider,
  model,
  onChange,
}: {
  options: ChatModelOption[]
  provider: string
  model: string
  onChange: (next: { provider: string; model: string }) => void
}) {
  const selected = modelChoiceKey(provider, model)
  return (
    <DropdownMenuContent
      side="top"
      align="start"
      className="menu-opaque w-max min-w-48 rounded-xl"
    >
      {options.map((o) => {
        const key = modelChoiceKey(o.provider, o.model)
        return (
          <DropdownMenuItem
            key={key}
            onClick={() => onChange({ provider: o.provider, model: o.model })}
            className="gap-2 text-xs font-normal"
          >
            {key === selected ? (
              <Check className="h-3.5 w-3.5 shrink-0" />
            ) : (
              <span className="h-3.5 w-3.5 shrink-0" />
            )}
            <span className="font-mono">{o.model}</span>
            <span className="text-muted-foreground">{o.providerLabel}</span>
          </DropdownMenuItem>
        )
      })}
    </DropdownMenuContent>
  )
}
