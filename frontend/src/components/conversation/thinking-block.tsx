import { useTranslation } from "react-i18next"
import { Brain, ChevronDown, ChevronRight } from "lucide-react"
import { Button } from "@/components/ui/button"
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible"
import { estimateTokens } from "@/lib/format"

interface ThinkingBlockProps {
  content: string
  isStreaming: boolean
  isExpanded: boolean
  onToggle: () => void
}

export function ThinkingBlock({
  content,
  isStreaming,
  isExpanded,
  onToggle,
}: ThinkingBlockProps) {
  const { t } = useTranslation()
  const tokens = estimateTokens(content)

  return (
    <Collapsible open={isExpanded} onOpenChange={onToggle}>
      <CollapsibleTrigger
        render={
          <Button
            variant="ghost"
            size="sm"
            className="h-auto gap-1 rounded-lg px-2 py-1 text-muted-foreground hover:bg-muted hover:text-foreground"
          />
        }
      >
        <Brain className="h-3 w-3" />
        {isExpanded ? (
          <ChevronDown className="h-3 w-3" />
        ) : (
          <ChevronRight className="h-3 w-3" />
        )}
        <span className="font-mono text-[10px]">
          {isStreaming
            ? t("status.thinking")
            : t("status.thinkingTokens", { tokens })}
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
