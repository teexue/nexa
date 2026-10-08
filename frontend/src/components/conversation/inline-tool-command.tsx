import { str, num } from "@/components/inspector/tool-detail-utils"
import { cn } from "@/lib/utils"
import { Terminal, type ToolRenderProps } from "./inline-tool-primitives"

export function RunCommand({ output }: ToolRenderProps) {
  const stdout = str(output?.stdout)
  const stderr = str(output?.stderr)
  const exitCode = num(output?.exit_code)
  if (!stdout && !stderr && exitCode === null) return null
  return (
    <Terminal>
      <Stream text={stdout} className="text-terminal-foreground" />
      <Stream text={stderr} className="text-terminal-error" />
      <ExitLine code={exitCode} timedOut={output?.timed_out === true} />
    </Terminal>
  )
}

function Stream({ text, className }: { text: string; className: string }) {
  if (!text) return null
  return (
    <pre className={cn("mt-1 break-all whitespace-pre-wrap", className)}>
      {text}
    </pre>
  )
}

function ExitLine({
  code,
  timedOut,
}: {
  code: number | null
  timedOut: boolean
}) {
  if (code === null) return null
  const tone = code === 0 ? "text-terminal-success" : "text-terminal-error"
  return (
    <div className="mt-1 flex items-center gap-1.5 text-terminal-muted">
      <span className={tone}>●</span>
      <span>exit {code}</span>
      {timedOut ? (
        <span className="text-terminal-error">· timed out</span>
      ) : null}
    </div>
  )
}
