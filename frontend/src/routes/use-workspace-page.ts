import { useCallback, useEffect, useMemo, useState } from "react"
import { useLocation, useNavigate } from "react-router"
import { useTheme } from "@/components/theme-provider"
import { useChat } from "@/hooks/use-chat"
import { useWorkspaceChat } from "./chat-context"
import { useAgentManager } from "@/hooks/use-agent-manager"
import { useKeyboardShortcuts } from "@/hooks/use-keyboard-shortcuts"
import { useMessageSearch } from "@/hooks/use-message-search"
import { deleteSession, fetchProviders } from "@/lib/api"
import type { ProviderInfo } from "@/types/agent"
import { useSessionStore } from "./session-store"
import {
  useWorkspaceMiscActions,
  useWorkspaceSend,
  useWorkspaceSessionActions,
} from "./use-workspace-actions"
import { useWorkspaceSessionSync } from "./use-workspace-session"
import { useWorkspaceAgent, useWorkspaceDerived } from "./use-workspace-state"

function useSessions(chat: ReturnType<typeof useChat>) {
  const { sessions, refresh } = useSessionStore()
  const removeSession = useCallback(
    async (id: string) => {
      try {
        await deleteSession(id)
        if (chat.sessionId === id) chat.clear()
        await refresh()
      } catch (err) {
        console.error("Failed to delete session:", err)
      }
    },
    [chat, refresh]
  )
  return useMemo(
    () => ({ sessions, refresh, remove: removeSession }),
    [sessions, refresh, removeSession]
  )
}

function useWorkspaceUi() {
  const [selectedToolCallId, setSelectedToolCallId] = useState<string | null>(
    null
  )
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false)
  const [sessionWorkDir, setSessionWorkDir] = useState<string | null>(null)
  const [providers, setProviders] = useState<ProviderInfo[]>([])
  const [runModel, setRunModel] = useState({ provider: "", model: "" })
  const [thinkEffort, setThinkEffort] = useState("")
  const workDir = (sessionWorkDir ?? localStorage.getItem("workDir")) || ""
  return {
    selectedToolCallId,
    setSelectedToolCallId,
    sidebarCollapsed,
    setSidebarCollapsed,
    sessionWorkDir,
    setSessionWorkDir,
    providers,
    setProviders,
    runModel,
    setRunModel,
    thinkEffort,
    setThinkEffort,
    workDir,
  }
}

export function useWorkspacePage() {
  const chat = useWorkspaceChat()
  const { theme, setTheme } = useTheme()
  const location = useLocation()
  const navigate = useNavigate()
  const sessMgr = useSessions(chat)
  const search = useMessageSearch(chat.messages)
  const agentMgr = useAgentManager()
  const ui = useWorkspaceUi()
  const agent = useWorkspaceAgent(agentMgr.agents, location.pathname)
  const derived = useWorkspaceDerived(
    chat,
    agent.agentInfo,
    ui.providers,
    ui.runModel
  )
  useWorkspaceSideEffects({ chat, location, navigate, sessMgr, ui, agent })
  const actions = useWorkspaceBoundActions({
    chat,
    navigate,
    sessMgr,
    ui,
    agent,
    agents: agentMgr.agents,
  })
  return {
    chat,
    theme,
    setTheme,
    navigate,
    sessMgr,
    search,
    agentMgr,
    ui,
    agent,
    derived,
    actions,
  }
}

function useWorkspaceSideEffects(ctx: {
  chat: ReturnType<typeof useChat>
  location: ReturnType<typeof useLocation>
  navigate: ReturnType<typeof useNavigate>
  sessMgr: ReturnType<typeof useSessions>
  ui: ReturnType<typeof useWorkspaceUi>
  agent: ReturnType<typeof useWorkspaceAgent>
}) {
  const { chat, location, navigate, sessMgr, ui, agent } = ctx
  const setProviders = ui.setProviders
  useEffect(() => {
    fetchProviders()
      .then((d) => setProviders(d ?? []))
      .catch(() => {})
  }, [setProviders])
  const refreshSessions = sessMgr.refresh
  useEffect(() => {
    if (chat.sessionId) refreshSessions()
  }, [chat.sessionId, refreshSessions])
  useResumeSync(chat, location, navigate, agent.setAgent, ui)
  useKeyboardShortcuts({
    onToggleSidebar: () => ui.setSidebarCollapsed((v) => !v),
    onClosePanel: () => ui.setSelectedToolCallId(null),
  })
}

function useResumeSync(
  chat: ReturnType<typeof useChat>,
  location: ReturnType<typeof useLocation>,
  navigate: ReturnType<typeof useNavigate>,
  setAgent: (id: string) => void,
  ui: ReturnType<typeof useWorkspaceUi>
) {
  const setSessionWorkDir = ui.setSessionWorkDir
  const setSelectedToolCallId = ui.setSelectedToolCallId
  const setRunModel = ui.setRunModel
  const onResumed = useCallback(
    (r: {
      agent: string
      workdir: string | null
      model: string
      provider: string
    }) => {
      setAgent(r.agent)
      setSessionWorkDir(r.workdir)
      setSelectedToolCallId(null)
      setRunModel({ provider: r.provider, model: r.model })
    },
    [setAgent, setSessionWorkDir, setSelectedToolCallId, setRunModel]
  )
  useWorkspaceSessionSync({
    sessionId: chat.sessionId,
    resumeSession: chat.resumeSession,
    locationSearch: location.search,
    locationPathname: location.pathname,
    navigate,
    onResumed,
  })
}

function useWorkspaceBoundActions(ctx: {
  chat: ReturnType<typeof useChat>
  navigate: ReturnType<typeof useNavigate>
  sessMgr: ReturnType<typeof useSessions>
  ui: ReturnType<typeof useWorkspaceUi>
  agent: ReturnType<typeof useWorkspaceAgent>
  agents: ReturnType<typeof useAgentManager>["agents"]
}) {
  const { chat, navigate, sessMgr, ui, agent, agents } = ctx
  const opts = {
    chat,
    navigate,
    agents,
    resolvedAgent: agent.resolvedAgent,
    agentInfo: agent.agentInfo,
    workDir: ui.workDir,
    runModel: {
      provider: ui.runModel.provider || agent.agentInfo?.provider || "",
      model: ui.runModel.model || agent.agentInfo?.model || "",
    },
    providers: ui.providers,
    thinkEffort: ui.thinkEffort,
    setRunModel: ui.setRunModel,
    refreshSessions: sessMgr.refresh,
    removeSession: sessMgr.remove,
    setAgent: agent.setAgent,
    setSelectedToolCallId: ui.setSelectedToolCallId,
    setSessionWorkDir: ui.setSessionWorkDir,
  }
  const session = useWorkspaceSessionActions(opts)
  const misc = useWorkspaceMiscActions(opts)
  const handleSendMessage = useWorkspaceSend(opts)
  return {
    ...session,
    ...misc,
    handleSendMessage,
    setRunModel: ui.setRunModel,
    setThinkEffort: ui.setThinkEffort,
  }
}
