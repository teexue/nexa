import { useTranslation } from "react-i18next"
import { Input } from "@/components/ui/input"
import type { AgentFormData } from "@/lib/agent-yaml"
import { Field, SectionCard } from "./shared"
import { RuntimeSelects } from "./runtime-selects"

export function RuntimeTab({
  form,
  setForm,
  knowledgeBases = [],
}: {
  form: AgentFormData
  setForm: React.Dispatch<React.SetStateAction<AgentFormData>>
  knowledgeBases?: Array<{ id: string; name: string }>
}) {
  return (
    <div className="space-y-4">
      <RuntimeLimitsCard form={form} setForm={setForm} />
      <RuntimeKnowledgeCard
        form={form}
        setForm={setForm}
        knowledgeBases={knowledgeBases}
      />
      <RuntimeOptimizeCard form={form} setForm={setForm} />
    </div>
  )
}

function RuntimeLimitsCard({
  form,
  setForm,
}: {
  form: AgentFormData
  setForm: React.Dispatch<React.SetStateAction<AgentFormData>>
}) {
  const { t } = useTranslation()
  return (
    <SectionCard
      title={t("agent.sectionRuntime")}
      description={t("agent.sectionRuntimeDesc")}
    >
      <div className="grid gap-4 sm:grid-cols-2">
        <Field label={t("agent.maxTurns")} hint={t("agent.maxTurnsHint")}>
          <Input
            type="number"
            value={form.maxTurns}
            onChange={(e) =>
              setForm((f) => ({ ...f, maxTurns: Number(e.target.value) }))
            }
            className="h-9 rounded-xl font-mono text-sm"
            min={0}
            max={10000}
          />
        </Field>
        <RuntimeSelects form={form} setForm={setForm} />
        <Field label={t("agent.maxParallel")} hint={t("agent.maxParallelHint")}>
          <Input
            type="number"
            value={form.maxParallel}
            onChange={(e) =>
              setForm((f) => ({ ...f, maxParallel: Number(e.target.value) }))
            }
            className="h-9 rounded-xl font-mono text-sm"
            min={1}
            max={16}
            disabled={form.execMode === "serial"}
          />
        </Field>
      </div>
    </SectionCard>
  )
}

function RuntimeKnowledgeCard({
  form,
  setForm,
  knowledgeBases,
}: {
  form: AgentFormData
  setForm: React.Dispatch<React.SetStateAction<AgentFormData>>
  knowledgeBases: Array<{ id: string; name: string }>
}) {
  const { t } = useTranslation()
  return (
    <SectionCard
      title={t("agent.sectionKnowledge")}
      description={t("agent.sectionKnowledgeDesc")}
    >
      <Field
        label={t("agent.knowledgeTopK")}
        hint={t("agent.knowledgeTopKHint")}
      >
        <Input
          type="number"
          value={form.knowledgeTopK}
          onChange={(e) =>
            setForm((f) => ({ ...f, knowledgeTopK: Number(e.target.value) }))
          }
          className="h-9 rounded-xl font-mono text-sm"
          min={1}
          max={20}
        />
      </Field>
      <KnowledgeBaseList
        form={form}
        setForm={setForm}
        knowledgeBases={knowledgeBases}
      />
    </SectionCard>
  )
}

function KnowledgeBaseList({
  form,
  setForm,
  knowledgeBases,
}: {
  form: AgentFormData
  setForm: React.Dispatch<React.SetStateAction<AgentFormData>>
  knowledgeBases: Array<{ id: string; name: string }>
}) {
  const { t } = useTranslation()
  if (knowledgeBases.length === 0) {
    return (
      <p className="mt-3 text-[11px] text-muted-foreground">
        {t("agent.knowledgeEmpty")}
      </p>
    )
  }
  return (
    <div className="mt-3 space-y-2">
      {knowledgeBases.map((kb) => (
        <KnowledgeBaseToggle
          key={kb.id}
          kb={kb}
          selected={form.knowledgeBases.includes(kb.id)}
          onToggle={() => setForm((prev) => toggleKb(prev, kb.id))}
        />
      ))}
    </div>
  )
}

function KnowledgeBaseToggle({
  kb,
  selected,
  onToggle,
}: {
  kb: { id: string; name: string }
  selected: boolean
  onToggle: () => void
}) {
  const { t } = useTranslation()
  return (
    <button
      type="button"
      onClick={onToggle}
      className={`flex w-full items-center justify-between rounded-xl border px-3 py-2.5 text-left text-xs transition-colors ${
        selected
          ? "border-primary/45 bg-primary/12"
          : "glass-tile border-[color:var(--glass-edge)] hover:bg-primary/12"
      }`}
    >
      <span>
        <span className="font-medium text-foreground">{kb.name}</span>
        <span className="ml-2 font-mono text-[10px] text-muted-foreground">
          {kb.id}
        </span>
      </span>
      <span className="text-[10px] text-muted-foreground">
        {selected ? t("common.selected") : t("common.select")}
      </span>
    </button>
  )
}

function RuntimeOptimizeCard({
  form,
  setForm,
}: {
  form: AgentFormData
  setForm: React.Dispatch<React.SetStateAction<AgentFormData>>
}) {
  const { t } = useTranslation()
  return (
    <SectionCard
      title={t("agent.sectionOptimize")}
      description={t("agent.sectionOptimizeDesc")}
    >
      <label className="flex cursor-pointer items-start gap-3">
        <input
          type="checkbox"
          checked={form.optimizeUserPrompt}
          onChange={(e) =>
            setForm((f) => ({ ...f, optimizeUserPrompt: e.target.checked }))
          }
          className="mt-0.5 h-3.5 w-3.5 rounded border-border accent-primary"
        />
        <span>
          <span className="block text-xs font-medium text-foreground">
            {t("agent.optimizeUserPrompt")}
          </span>
          <span className="mt-0.5 block text-[11px] leading-relaxed text-muted-foreground">
            {t("agent.optimizeUserPromptHint")}
          </span>
        </span>
      </label>
    </SectionCard>
  )
}

function toggleKb(prev: AgentFormData, id: string): AgentFormData {
  const selected = prev.knowledgeBases.includes(id)
  let tools = prev.tools
  let knowledgeBases = prev.knowledgeBases
  if (selected) {
    knowledgeBases = knowledgeBases.filter((x) => x !== id)
  } else {
    knowledgeBases = [...knowledgeBases, id]
    if (!tools.includes("knowledge_search"))
      tools = [...tools, "knowledge_search"]
    if (!tools.includes("knowledge_list")) tools = [...tools, "knowledge_list"]
  }
  return { ...prev, knowledgeBases, tools }
}
