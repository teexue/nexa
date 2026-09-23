import { useTranslation } from "react-i18next"
import type { AgentFormData } from "@/lib/agent-yaml"
import { toolDisplayDescription, toolDisplayName } from "@/lib/tool-i18n"
import type { ToolInfo } from "@/types/agent"
import {
  getPermConfig,
  getToolPermission,
  setToolPermission,
  type ToolPermission,
} from "./perm-utils"
import { toggleToolInForm } from "./tools-selection"

export function ToolRow({
  tool,
  form,
  setForm,
}: {
  tool: ToolInfo
  form: AgentFormData
  setForm: React.Dispatch<React.SetStateAction<AgentFormData>>
}) {
  const { t } = useTranslation()
  const selected = form.tools.includes(tool.name)
  const perm = getToolPermission(form, tool.name)
  const permConfig = getPermConfig(t)
  return (
    <div
      className={`rounded-xl border px-3 py-2.5 transition-colors ${selected ? "border-primary/45 bg-primary/12" : "glass-tile border-[color:var(--glass-edge)] hover:bg-primary/12"}`}
    >
      <label className="flex cursor-pointer items-start gap-3">
        <input
          type="checkbox"
          checked={selected}
          onChange={() => setForm((prev) => toggleToolInForm(prev, tool.name))}
          className="mt-1 h-3.5 w-3.5 rounded border-border accent-primary"
        />
        <div className="min-w-0 flex-1">
          <p className="text-xs font-medium text-foreground">
            {toolDisplayName(tool.name, t)}
          </p>
          <p className="mt-0.5 font-mono text-[10px] text-muted-foreground">
            {tool.name}
          </p>
          <p className="mt-0.5 line-clamp-2 text-[11px] leading-relaxed text-muted-foreground">
            {toolDisplayDescription(tool.name, tool.description, t)}
          </p>
        </div>
      </label>
      {selected && (
        <PermButtons
          perm={perm}
          permConfig={permConfig}
          onPick={(p) => setForm((f) => setToolPermission(f, tool.name, p))}
        />
      )}
    </div>
  )
}

function PermButtons({
  perm,
  permConfig,
  onPick,
}: {
  perm: ToolPermission
  permConfig: ReturnType<typeof getPermConfig>
  onPick: (p: ToolPermission) => void
}) {
  return (
    <div className="mt-2 flex flex-wrap gap-1 border-t border-border/60 pt-2 pl-6">
      {(Object.keys(permConfig) as ToolPermission[]).map((p) => {
        const c = permConfig[p]
        const active = perm === p
        return (
          <button
            key={p}
            type="button"
            onClick={() => onPick(p)}
            className={`flex items-center gap-1 rounded-md px-2 py-1 text-[10px] transition-colors ${
              active
                ? `${c.bg} ${c.color} font-medium`
                : "text-muted-foreground hover:bg-muted"
            }`}
          >
            <c.icon className="h-3 w-3" /> {c.label}
          </button>
        )
      })}
    </div>
  )
}
