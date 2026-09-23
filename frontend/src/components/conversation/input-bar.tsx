import { useTranslation } from "react-i18next"
import { Textarea } from "@/components/ui/textarea"
import { useInputBar } from "@/hooks/use-input-bar"
import { AttachFilesButton, InputBarFooterRight } from "./input-bar-actions"
import { InputBarAttachments } from "./input-bar-attachments"
import {
  TokenUsageIndicator,
  type SessionTokenUsage,
} from "./token-usage-indicator"
import type { FileAttachment } from "@/types/agent"

interface InputBarProps {
  onSend: (text: string, attachments: FileAttachment[]) => void
  onStop?: () => void
  onOptimize?: (text: string) => Promise<string>
  disabled: boolean
  isStreaming?: boolean
  visionEnabled?: boolean
  optimizing?: boolean
  accessory?: React.ReactNode
  tokenUsage?: SessionTokenUsage
}

export function InputBar(props: InputBarProps) {
  const {
    text,
    setText,
    attachments,
    fileInputRef,
    handleFileSelect,
    removeAttachment,
    handleOptimize,
    handleSend,
    handleKeyDown,
  } = useInputBar(props)
  return (
    <div className="shrink-0 px-5 pt-2 pb-5">
      <InputBarAttachments
        attachments={attachments}
        onRemove={removeAttachment}
      />
      <div className="glass-tile relative rounded-xl border border-[color:var(--glass-edge)] transition-shadow focus-within:border-primary/45">
        <PromptField
          text={text}
          onChange={setText}
          onKeyDown={handleKeyDown}
          isStreaming={!!props.isStreaming}
          fileInputRef={fileInputRef}
          onFileSelect={handleFileSelect}
        />
        <InputBarToolbar
          accessory={props.accessory}
          tokenUsage={props.tokenUsage}
          isStreaming={!!props.isStreaming}
          showOptimize={!!props.onOptimize}
          optimizing={props.optimizing}
          optimizeDisabled={
            props.disabled || !text.trim() || !!props.optimizing
          }
          sendDisabled={
            props.disabled || (!text.trim() && attachments.length === 0)
          }
          onOptimizeClick={handleOptimize}
          onSend={handleSend}
          onStop={props.onStop}
        />
      </div>
    </div>
  )
}

function PromptField({
  text,
  onChange,
  onKeyDown,
  isStreaming,
  fileInputRef,
  onFileSelect,
}: {
  text: string
  onChange: (v: string) => void
  onKeyDown: (e: React.KeyboardEvent) => void
  isStreaming: boolean
  fileInputRef: React.RefObject<HTMLInputElement | null>
  onFileSelect: (e: React.ChangeEvent<HTMLInputElement>) => void
}) {
  const { t } = useTranslation()
  return (
    <div className="flex items-start gap-0.5 pt-2.5 pr-3 pb-1 pl-2">
      <AttachFilesButton onClick={() => fileInputRef.current?.click()} />
      <input
        ref={fileInputRef}
        type="file"
        multiple
        className="hidden"
        onChange={onFileSelect}
      />
      <Textarea
        value={text}
        onChange={(e) => onChange(e.target.value)}
        onKeyDown={onKeyDown}
        placeholder={
          isStreaming
            ? t("conversation.placeholderStreaming")
            : t("conversation.placeholderIdle")
        }
        disabled={false}
        rows={1}
        className="glass-plain field-sizing-content max-h-[6lh] min-h-6 flex-1 resize-none overflow-y-auto overscroll-contain border-0 bg-transparent px-1.5 py-0 text-sm leading-6 shadow-none focus-visible:ring-0"
      />
    </div>
  )
}

function InputBarToolbar({
  accessory,
  tokenUsage,
  isStreaming,
  showOptimize,
  optimizing,
  optimizeDisabled,
  sendDisabled,
  onOptimizeClick,
  onSend,
  onStop,
}: {
  accessory?: React.ReactNode
  tokenUsage?: SessionTokenUsage
  isStreaming: boolean
  showOptimize: boolean
  optimizing?: boolean
  optimizeDisabled: boolean
  sendDisabled: boolean
  onOptimizeClick: () => void
  onSend: () => void
  onStop?: () => void
}) {
  return (
    <div className="flex flex-nowrap items-center gap-2 px-2 pb-2">
      <div className="flex min-w-0 flex-1 items-center overflow-hidden">
        {accessory}
      </div>
      {tokenUsage ? (
        <div className="shrink-0">
          <TokenUsageIndicator usage={tokenUsage} />
        </div>
      ) : null}
      <InputBarFooterRight
        isStreaming={isStreaming}
        showOptimize={showOptimize}
        optimizing={optimizing}
        optimizeDisabled={optimizeDisabled}
        sendDisabled={sendDisabled}
        onOptimizeClick={onOptimizeClick}
        onSend={onSend}
        onStop={onStop}
      />
    </div>
  )
}
