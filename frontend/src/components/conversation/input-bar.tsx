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
      <div className="composer-surface glass-tile relative rounded-xl">
        <PromptField
          text={text}
          onChange={setText}
          onKeyDown={handleKeyDown}
          isStreaming={!!props.isStreaming}
        />
        <InputBarToolbar
          accessory={
            <ComposerLeading
              fileInputRef={fileInputRef}
              onFileSelect={handleFileSelect}
              accessory={props.accessory}
            />
          }
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
}: {
  text: string
  onChange: (v: string) => void
  onKeyDown: (e: React.KeyboardEvent) => void
  isStreaming: boolean
}) {
  const { t } = useTranslation()
  return (
    <div className="px-3 pt-2.5 pb-1">
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
        className="glass-plain field-sizing-content max-h-[6lh] min-h-6 w-full resize-none overflow-y-auto overscroll-contain border-0 bg-transparent px-0.5 py-0 text-sm leading-6 shadow-none focus-visible:ring-0"
      />
    </div>
  )
}

function ComposerLeading({
  fileInputRef,
  onFileSelect,
  accessory,
}: {
  fileInputRef: React.RefObject<HTMLInputElement | null>
  onFileSelect: (e: React.ChangeEvent<HTMLInputElement>) => void
  accessory?: React.ReactNode
}) {
  return (
    <>
      <AttachFilesButton onClick={() => fileInputRef.current?.click()} />
      <input
        ref={fileInputRef}
        type="file"
        multiple
        className="hidden"
        onChange={onFileSelect}
      />
      {accessory}
    </>
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
      <div className="flex min-w-0 flex-1 items-center gap-0.5 overflow-hidden">
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
