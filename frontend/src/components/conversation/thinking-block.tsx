import { useState } from "react"
import { useTranslation } from "react-i18next"
import { Brain, ChevronDown, ChevronRight } from "lucide-react"
import { Button } from "@/components/ui/button"
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible"
import { latestTextLine } from "@/lib/activity-fold"
import { estimateTokens } from "@/lib/format"

interface ThinkingBlockProps {
  content: string
  isStreaming: boolean
}

export function ThinkingBlock({ content, isStreaming }: ThinkingBlockProps) {
  const [open, setOpen] = useState(false)
  const { t } = useTranslation()
  const line = latestTextLine(content)
  const tokens = estimateTokens(content)
  const label =
    line ||
    (isStreaming
      ? t("status.thinking")
      : t("status.thinkingTokens", { tokens }))

  return (
    <Collapsible open={open} onOpenChange={setOpen}>
      <CollapsibleTrigger
        render={
          <Button
            variant="ghost"
            size="sm"
            className="h-auto max-w-full gap-1 rounded-lg px-2 py-1 text-muted-foreground hover:bg-muted hover:text-foreground"
          />
        }
      >
        <Brain className="h-3 w-3 shrink-0" />
        {open ? (
          <ChevronDown className="h-3 w-3 shrink-0" />
        ) : (
          <ChevronRight className="h-3 w-3 shrink-0" />
        )}
        <span className="min-w-0 truncate font-mono text-[11px]" title={label}>
          {open ? t("status.thinkingTokens", { tokens }) : label}
        </span>
      </CollapsibleTrigger>
      <CollapsibleContent>
        <div className="mt-1 ml-3 border-l-2 border-primary/15 pl-3">
          <p className="text-xs leading-relaxed wrap-anywhere whitespace-pre-wrap text-muted-foreground">
            {content}
          </p>
        </div>
      </CollapsibleContent>
    </Collapsible>
  )
}
