import { useTranslation } from "react-i18next"
import { Plug, Trash2 } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import type { McpServerFormItem } from "@/lib/agent-yaml"
import { Field } from "./shared"
import { McpTypeSelect } from "./mcp-type-select"

export function McpServerRow({
  server,
  onChange,
  onRemove,
}: {
  server: McpServerFormItem
  onChange: (patch: Partial<McpServerFormItem>) => void
  onRemove: () => void
}) {
  const { t } = useTranslation()
  return (
    <div className="glass-tile rounded-xl border border-[color:var(--glass-edge)] p-4">
      <div className="mb-3 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Plug className="h-3.5 w-3.5 text-muted-foreground" />
          <span className="text-xs font-medium text-foreground">
            {server.name || t("agent.mcpUntitled")}
          </span>
        </div>
        <Button
          variant="ghost"
          size="icon-xs"
          className="h-7 w-7 rounded-lg text-muted-foreground hover:text-destructive"
          onClick={onRemove}
        >
          <Trash2 className="h-3.5 w-3.5" />
        </Button>
      </div>
      <div className="grid gap-3 sm:grid-cols-2">
        <Field label={t("agent.mcpName")} hint={t("agent.mcpNameHint")}>
          <Input
            value={server.name}
            onChange={(e) => onChange({ name: e.target.value })}
            placeholder="filesystem"
            className="h-9 rounded-xl font-mono text-sm"
          />
        </Field>
        <McpTypeSelect server={server} onChange={onChange} />
      </div>
      <McpTransportFields server={server} onChange={onChange} />
      <Field label={t("agent.mcpEnv")} hint={t("agent.mcpEnvHint")}>
        <Textarea
          value={server.env}
          onChange={(e) => onChange({ env: e.target.value })}
          placeholder={"NODE_ENV=production\nAPI_KEY=xxx"}
          className="mt-1 min-h-16 resize-y rounded-xl font-mono text-xs leading-relaxed"
        />
      </Field>
    </div>
  )
}

function McpTransportFields({
  server,
  onChange,
}: {
  server: McpServerFormItem
  onChange: (patch: Partial<McpServerFormItem>) => void
}) {
  const { t } = useTranslation()
  if (server.type !== "stdio") {
    return (
      <Field label={t("agent.mcpUrl")} hint={t("agent.mcpUrlHint")}>
        <Input
          value={server.url}
          onChange={(e) => onChange({ url: e.target.value })}
          placeholder="https://example.com/mcp/sse"
          className="mt-1 h-9 rounded-xl font-mono text-sm"
        />
      </Field>
    )
  }
  return (
    <>
      <Field label={t("agent.mcpCommand")} hint={t("agent.mcpCommandHint")}>
        <Input
          value={server.command}
          onChange={(e) => onChange({ command: e.target.value })}
          placeholder="npx"
          className="mt-1 h-9 rounded-xl font-mono text-sm"
        />
      </Field>
      <Field label={t("agent.mcpArgs")} hint={t("agent.mcpArgsHint")}>
        <Textarea
          value={server.args}
          onChange={(e) => onChange({ args: e.target.value })}
          placeholder={"-y\n@modelcontextprotocol/server-filesystem\n/tmp"}
          className="mt-1 min-h-20 resize-y rounded-xl font-mono text-xs leading-relaxed"
        />
      </Field>
    </>
  )
}
