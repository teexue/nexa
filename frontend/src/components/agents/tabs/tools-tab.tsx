import { useMemo, useState } from "react"
import { useTranslation } from "react-i18next"
import { Search } from "lucide-react"
import { EmptyState } from "@/components/shared/empty-state"
import { Badge } from "@/components/ui/badge"
import { Input } from "@/components/ui/input"
import type { AgentFormData } from "@/lib/agent-yaml"
import type { ToolInfo } from "@/types/agent"
import { SectionCard } from "./shared"
import { ToolRow } from "./tool-row"
import { filterTools, selectAllPatch } from "./tools-selection"

export function ToolsTab({
  form,
  setForm,
  tools,
}: {
  form: AgentFormData
  setForm: React.Dispatch<React.SetStateAction<AgentFormData>>
  tools: ToolInfo[]
}) {
  const { t } = useTranslation()
  const [query, setQuery] = useState("")
  const filtered = useMemo(
    () => filterTools(tools, query, t),
    [tools, query, t]
  )
  const filteredNames = useMemo(() => filtered.map((x) => x.name), [filtered])
  const selectedFiltered = filteredNames.filter((n) => form.tools.includes(n))
  const allSelected =
    filteredNames.length > 0 && selectedFiltered.length === filteredNames.length

  return (
    <div className="space-y-4">
      <SectionCard
        title={t("agent.sectionTools")}
        description={t("agent.sectionToolsDesc")}
      >
        <ToolsToolbar
          query={query}
          onQuery={setQuery}
          allSelected={allSelected}
          someSelected={selectedFiltered.length > 0}
          filteredCount={filtered.length}
          selectedCount={form.tools.length}
          onToggleAll={() =>
            setForm((prev) => selectAllPatch(prev, filteredNames, allSelected))
          }
        />
        {filtered.length === 0 ? (
          <EmptyState title={t("agent.noToolsMatch")} />
        ) : (
          <div className="max-h-[28rem] space-y-1.5 overflow-y-auto pr-1">
            {filtered.map((tool) => (
              <ToolRow
                key={tool.name}
                tool={tool}
                form={form}
                setForm={setForm}
              />
            ))}
          </div>
        )}
      </SectionCard>
    </div>
  )
}

function ToolsToolbar({
  query,
  onQuery,
  allSelected,
  someSelected,
  filteredCount,
  selectedCount,
  onToggleAll,
}: {
  query: string
  onQuery: (q: string) => void
  allSelected: boolean
  someSelected: boolean
  filteredCount: number
  selectedCount: number
  onToggleAll: () => void
}) {
  const { t } = useTranslation()
  return (
    <div className="flex items-center gap-2">
      <label className="glass-tile flex cursor-pointer items-center gap-1.5 rounded-lg border border-[color:var(--glass-edge)] px-2.5 py-1.5 text-[11px] text-muted-foreground transition-colors hover:bg-primary/12">
        <input
          type="checkbox"
          ref={(el) => {
            if (el) el.indeterminate = someSelected && !allSelected
          }}
          checked={allSelected}
          onChange={onToggleAll}
          disabled={filteredCount === 0}
          className="h-3.5 w-3.5 rounded border-border accent-primary"
        />
        {allSelected ? t("agent.deselectAll") : t("agent.selectAll")}
      </label>
      <div className="relative flex-1">
        <Search className="pointer-events-none absolute top-1/2 left-2.5 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
        <Input
          value={query}
          onChange={(e) => onQuery(e.target.value)}
          placeholder={t("agent.searchTools")}
          className="h-9 rounded-xl pl-8 text-sm"
        />
      </div>
      <Badge variant="secondary" className="rounded-md px-2 py-1 text-[10px]">
        {t("agent.selectedCount", { count: selectedCount })}
      </Badge>
    </div>
  )
}
