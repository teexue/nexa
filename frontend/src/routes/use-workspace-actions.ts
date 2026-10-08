import { useCallback } from "react"
import type { NavigateFunction } from "react-router"
import type { AgentInfo, FileAttachment, ProviderInfo } from "@/types/agent"
import { effortForProvider } from "@/lib/think-effort"
import type { useChat } from "@/hooks/use-chat"
import { resolveApproval, updateSessionWorkdir } from "@/lib/api"

interface WorkspaceActionOpts {
  chat: ReturnType<typeof useChat>
  navigate: NavigateFunction
  agents: AgentInfo[]
  resolvedAgent: string
  agentInfo: AgentInfo | null
  workDir: string
  runModel: { provider: string; model: string }
  providers: ProviderInfo[]
  thinkEffort: string
  setRunModel: (v: { provider: string; model: string }) => void
  refreshSessions: () => void
  removeSession: (id: string) => void
  setAgent: (id: string) => void
  setSelectedToolCallId: (
    id: string | null | ((p: string | null) => string | null)
  ) => void
  setSessionWorkDir: (dir: string | null) => void
}

export function useWorkspaceSend(
  opts: Pick<
    WorkspaceActionOpts,
    | "chat"
    | "agentInfo"
    | "resolvedAgent"
    | "workDir"
    | "runModel"
    | "providers"
    | "thinkEffort"
  >
) {
  return useCallback(
    (text: string, attachments?: FileAttachment[]) =>
      opts.chat.sendMessage({
        text,
        agent:
          opts.agentInfo?.id ||
          opts.resolvedAgent ||
          opts.agentInfo?.name ||
          "",
        workDir: opts.workDir || undefined,
        attachments,
        model: opts.runModel.model || undefined,
        provider: opts.runModel.provider || undefined,
        thinkingEffort: sendEffort(opts),
      }),
    [opts]
  )
}

function sendEffort(
  opts: Pick<WorkspaceActionOpts, "providers" | "runModel" | "thinkEffort">
): string | undefined {
  const effort = effortForProvider(
    opts.providers,
    opts.runModel.provider,
    opts.runModel.model,
    opts.thinkEffort
  )
  return effort || undefined
}

function applyResumedSession(
  opts: WorkspaceActionOpts,
  id: string,
  r: { agent: string; workdir: string | null; model: string; provider: string }
) {
  const ref =
    opts.agents.find((a) => a.id === r.agent || a.name === r.agent)?.id ??
    r.agent
  opts.setAgent(ref)
  opts.setSessionWorkDir(r.workdir)
  opts.setRunModel({ provider: r.provider, model: r.model })
  opts.setSelectedToolCallId(null)
  const params = new URLSearchParams()
  params.set("session", id)
  opts.navigate(`/agents/${encodeURIComponent(ref)}?${params.toString()}`, {
    replace: true,
  })
}

export function useWorkspaceSessionActions(opts: WorkspaceActionOpts) {
  const handleNewSession = useCallback(() => {
    opts.chat.clear()
    opts.setSelectedToolCallId(null)
    opts.setSessionWorkDir(null)
    opts.setRunModel({ provider: "", model: "" })
    opts.refreshSessions()
    const path = opts.resolvedAgent
      ? `/agents/${encodeURIComponent(opts.resolvedAgent)}`
      : "/"
    opts.navigate(path, { replace: true })
  }, [opts])
  const handleResumeSession = useCallback(
    async (id: string) => {
      const r = await opts.chat.resumeSession(id)
      if (r) applyResumedSession(opts, id, r)
    },
    [opts]
  )
  const handleWorkdirChange = useCallback(
    async (dir: string) => {
      opts.setSessionWorkDir(dir || null)
      if (opts.chat.sessionId) {
        try {
          await updateSessionWorkdir(opts.chat.sessionId, dir)
        } catch (e) {
          console.error(e)
        }
      }
    },
    [opts]
  )
  const handleDeleteSession = useCallback(
    (id: string) => {
      if (id === opts.chat.sessionId) opts.setSessionWorkDir(null)
      opts.removeSession(id)
    },
    [opts]
  )
  return {
    handleNewSession,
    handleResumeSession,
    handleWorkdirChange,
    handleDeleteSession,
  }
}

export function useWorkspaceMiscActions(
  opts: Pick<
    WorkspaceActionOpts,
    "chat" | "navigate" | "setAgent" | "setSelectedToolCallId" | "setRunModel"
  >
) {
  const handleSelectToolCall = useCallback(
    (id: string) => {
      opts.setSelectedToolCallId((p) => (p === id ? null : id))
    },
    [opts]
  )
  const handleSelectAgent = useCallback(
    (id: string) => {
      if (opts.chat.messages.length > 0) return
      opts.setAgent(id)
      opts.setRunModel({ provider: "", model: "" })
      opts.setSelectedToolCallId(null)
      opts.navigate(`/agents/${encodeURIComponent(id)}`, { replace: true })
    },
    [opts]
  )
  const resolveToolApproval = useCallback(
    async (id: string, approve: boolean) => {
      try {
        await resolveApproval(id, approve)
      } catch (e) {
        console.error(e)
      }
    },
    []
  )
  return { handleSelectToolCall, handleSelectAgent, resolveToolApproval }
}
