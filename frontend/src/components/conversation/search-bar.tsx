import { useEffect, useRef, useState } from "react"
import { useTranslation } from "react-i18next"
import { ChevronDown, ChevronUp, Search, X } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { isComposingEvent } from "@/lib/keys"

interface SearchBarProps {
  onSearch: (query: string) => void
  onClear: () => void
  matchCount: number
  currentMatch: number
  onPrev: () => void
  onNext: () => void
}

function MatchNav({
  matchCount,
  currentMatch,
  onPrev,
  onNext,
}: Pick<SearchBarProps, "matchCount" | "currentMatch" | "onPrev" | "onNext">) {
  const { t } = useTranslation()
  return (
    <>
      <span className="text-[10px] whitespace-nowrap text-muted-foreground">
        {matchCount > 0
          ? `${currentMatch + 1}/${matchCount}`
          : t("conversation.noMatch")}
      </span>
      <div className="flex items-center gap-0.5">
        <Button
          variant="ghost"
          size="icon-xs"
          className="h-5 w-5 rounded-md"
          onClick={onPrev}
          disabled={matchCount === 0}
        >
          <ChevronUp className="h-3 w-3" />
        </Button>
        <Button
          variant="ghost"
          size="icon-xs"
          className="h-5 w-5 rounded-md"
          onClick={onNext}
          disabled={matchCount === 0}
        >
          <ChevronDown className="h-3 w-3" />
        </Button>
      </div>
    </>
  )
}

export function SearchBar({ onSearch, ...nav }: SearchBarProps) {
  const { t } = useTranslation()
  const [query, setQuery] = useState("")
  const inputRef = useRef<HTMLInputElement>(null)
  useEffect(() => {
    inputRef.current?.focus()
  }, [])
  useEffect(() => {
    const timer = setTimeout(() => onSearch(query), 200)
    return () => clearTimeout(timer)
  }, [query, onSearch])
  return (
    <SearchBarRow
      query={query}
      setQuery={setQuery}
      inputRef={inputRef}
      placeholder={t("conversation.searchPlaceholder")}
      props={{ onSearch, ...nav }}
    />
  )
}

function SearchBarRow({
  query,
  setQuery,
  inputRef,
  placeholder,
  props,
}: {
  query: string
  setQuery: (q: string) => void
  inputRef: React.RefObject<HTMLInputElement | null>
  placeholder: string
  props: SearchBarProps
}) {
  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (isComposingEvent(e)) return
    if (e.key === "Enter") {
      e.preventDefault()
      if (e.shiftKey) props.onPrev()
      else props.onNext()
    }
    if (e.key === "Escape") props.onClear()
  }
  return (
    <div className="glass-tile flex items-center gap-2 rounded-xl border border-[color:var(--glass-edge)] px-3 py-2">
      <Search className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
      <Input
        ref={inputRef}
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        onKeyDown={handleKeyDown}
        placeholder={placeholder}
        className="h-6 flex-1 border-0 bg-transparent p-0 text-xs shadow-none focus-visible:ring-0"
      />
      {query && (
        <>
          <MatchNav {...props} />
          <Button
            variant="ghost"
            size="icon-xs"
            className="h-5 w-5 rounded-md"
            onClick={() => {
              setQuery("")
              props.onClear()
            }}
          >
            <X className="h-3 w-3" />
          </Button>
        </>
      )}
    </div>
  )
}
