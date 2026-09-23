import { useEffect, useMemo, useState } from "react"
import { useTranslation } from "react-i18next"
import { FileCode2 } from "lucide-react"
import { PageHeader } from "@/components/shared/page-header"
import { PageMain, PageShell } from "@/components/shared/page-shell"
import {
  buildRunSamples,
  buildTokenSamples,
} from "@/components/docs/api-docs-samples"
import {
  SECTIONS,
  useBaseURL,
  type SectionId,
} from "@/components/docs/api-docs-shared"
import { BaseURLChip } from "@/components/docs/api-docs-widgets"
import { DocsToc } from "@/components/docs/api-docs-toc"
import {
  ApproveSection,
  AuthSection,
  ErrorsSection,
  EventsSection,
  OverviewSection,
  RunSection,
  SessionSection,
} from "@/components/docs/api-docs-sections"
import { cn } from "@/lib/utils"

function useActiveSection() {
  const [active, setActive] = useState<SectionId>("overview")
  useEffect(() => {
    const nodes = SECTIONS.map((id) => document.getElementById(id)).filter(
      Boolean
    ) as HTMLElement[]
    if (nodes.length === 0) return
    const obs = new IntersectionObserver(
      (entries) => {
        const visible = entries
          .filter((e) => e.isIntersecting)
          .sort((a, b) => b.intersectionRatio - a.intersectionRatio)[0]
        if (visible?.target?.id) setActive(visible.target.id as SectionId)
      },
      { rootMargin: "-18% 0px -55% 0px", threshold: [0, 0.2, 0.5, 1] }
    )
    nodes.forEach((n) => obs.observe(n))
    return () => obs.disconnect()
  }, [])
  const scrollTo = (id: SectionId) => {
    document
      .getElementById(id)
      ?.scrollIntoView({ behavior: "smooth", block: "start" })
    setActive(id)
  }
  return { active, scrollTo }
}

function DocsMobileNav({
  active,
  onSelect,
}: {
  active: SectionId
  onSelect: (id: SectionId) => void
}) {
  const { t } = useTranslation()
  return (
    <div className="glass-panel sticky top-0 z-10 -mx-6 -mt-6 mb-6 flex gap-1.5 overflow-x-auto border-b border-[color:var(--glass-edge)] px-4 py-2 lg:hidden">
      {SECTIONS.map((id) => (
        <button
          key={id}
          type="button"
          onClick={() => onSelect(id)}
          className={cn(
            "shrink-0 rounded-lg px-2.5 py-1 text-xs transition-colors",
            active === id
              ? "bg-primary/10 font-medium text-primary"
              : "text-muted-foreground hover:bg-muted/50"
          )}
        >
          {t(`apiDocs.nav.${id}`)}
        </button>
      ))}
    </div>
  )
}

function DocsBody({
  base,
  runSamples,
  tokenSamples,
}: {
  base: string
  runSamples: ReturnType<typeof buildRunSamples>
  tokenSamples: ReturnType<typeof buildTokenSamples>
}) {
  return (
    <div className="w-full space-y-6">
      <OverviewSection />
      <AuthSection samples={tokenSamples} />
      <RunSection samples={runSamples} />
      <EventsSection />
      <ApproveSection base={base} />
      <SessionSection base={base} />
      <ErrorsSection />
    </div>
  )
}

/** Interactive API integration guide (not Markdown). */
export function ApiDocsPage() {
  const { t } = useTranslation()
  const base = useBaseURL()
  const { active, scrollTo } = useActiveSection()
  const runSamples = useMemo(() => buildRunSamples(base), [base])
  const tokenSamples = useMemo(
    () =>
      buildTokenSamples(
        base,
        t("apiDocs.sampleLogin"),
        t("apiDocs.sampleExchange")
      ),
    [base, t]
  )

  return (
    <PageShell>
      <PageHeader
        icon={FileCode2}
        title={t("apiDocs.title")}
        actions={<BaseURLChip url={base} />}
      />
      <div className="flex min-h-0 flex-1">
        <DocsToc active={active} onSelect={scrollTo} />
        <PageMain contentClassName="pb-16">
          <DocsMobileNav active={active} onSelect={scrollTo} />
          <DocsBody
            base={base}
            runSamples={runSamples}
            tokenSamples={tokenSamples}
          />
        </PageMain>
      </div>
    </PageShell>
  )
}
