import { AlertCircle } from "lucide-react"
import { ScrollArea } from "@/components/ui/scroll-area"
import { SearchBar } from "./search-bar"
import { StreamProgress } from "./stream-progress"
import { ActivityEntry } from "./activity-entry"
import type { MessageSearch } from "@/hooks/use-message-search"
import type { ConversationEntry } from "@/types/agent"
import { cn } from "@/lib/utils"

interface WorkspaceMessageListProps {
  messages: ConversationEntry[]
  isStreaming: boolean
  error: string | null
  selectedToolCallId: string | null
  onSelectToolCall: (id: string) => void
  onApproveTool?: (approvalId: string) => void
  onDenyTool?: (approvalId: string) => void
  search: MessageSearch
  containerRef: React.RefObject<HTMLDivElement | null>
  onScroll: () => void
}

export function WorkspaceMessageList({
  messages,
  isStreaming,
  error,
  selectedToolCallId,
  onSelectToolCall,
  onApproveTool,
  onDenyTool,
  search,
  containerRef,
  onScroll,
}: WorkspaceMessageListProps) {
  return (
    <ScrollArea className="h-full min-w-0">
      <div
        ref={containerRef}
        onScroll={onScroll}
        className="h-full min-w-0 overflow-x-hidden overflow-y-auto"
      >
        <div className="flex min-w-0 flex-col gap-2.5 px-5 py-4">
          {search.searchOpen && (
            <div className="glass-panel sticky top-0 z-10 -mx-5 px-5 py-2">
              <SearchBar
                onSearch={search.setSearchQuery}
                onClear={search.handleClear}
                matchCount={search.searchResults.length}
                currentMatch={search.currentMatch}
                onPrev={search.handlePrev}
                onNext={search.handleNext}
              />
            </div>
          )}
          <StreamProgress active={isStreaming} />
          {messages.map((entry, msgIndex) => (
            <MessageRow
              key={entry.id}
              entry={entry}
              msgIndex={msgIndex}
              isLast={msgIndex === messages.length - 1}
              isStreaming={isStreaming}
              selectedToolCallId={selectedToolCallId}
              onSelectToolCall={onSelectToolCall}
              onApproveTool={onApproveTool}
              onDenyTool={onDenyTool}
              search={search}
              matchesRef={search.matchRefs}
            />
          ))}
          {error && (
            <div className="flex items-center gap-2 rounded-xl border border-destructive/20 bg-destructive/5 px-3.5 py-2.5 text-xs text-destructive">
              <AlertCircle className="h-3.5 w-3.5 shrink-0" />
              <span>{error}</span>
            </div>
          )}
        </div>
      </div>
    </ScrollArea>
  )
}

function MessageRow({
  entry,
  msgIndex,
  isLast,
  isStreaming,
  selectedToolCallId,
  onSelectToolCall,
  onApproveTool,
  onDenyTool,
  search,
  matchesRef,
}: {
  entry: ConversationEntry
  msgIndex: number
  isLast: boolean
  isStreaming: boolean
  selectedToolCallId: string | null
  onSelectToolCall: (id: string) => void
  onApproveTool?: (approvalId: string) => void
  onDenyTool?: (approvalId: string) => void
  search: MessageSearch
  matchesRef: React.MutableRefObject<HTMLDivElement[]>
}) {
  const matchIdx = search.searchResults.findIndex((r) => r.index === msgIndex)
  const isCurrent =
    search.matchedIndices.has(msgIndex) && matchIdx === search.currentMatch
  return (
    <div
      ref={(el) => {
        if (el && matchIdx >= 0) matchesRef.current[matchIdx] = el
      }}
      className={cn(
        "max-w-full min-w-0",
        isCurrent && "rounded-xl ring-1 ring-primary/40"
      )}
    >
      <ActivityEntry
        entry={entry}
        selectedToolCallId={selectedToolCallId}
        onSelectToolCall={onSelectToolCall}
        onApproveTool={onApproveTool}
        onDenyTool={onDenyTool}
        isActive={isStreaming && isLast}
      />
    </div>
  )
}
