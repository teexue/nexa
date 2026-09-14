import { useCallback, useState } from "react"
import { useNavigate } from "react-router"
import { useKeyboardShortcuts } from "@/hooks/use-keyboard-shortcuts"
import { useAgentManager } from "@/hooks/use-agent-manager"
import { useSessionStore } from "./session-store"
import { useWorkspaceChat } from "./chat-context"
import { fetchSession } from "@/lib/api"

export function useSessionList() {
  const { sessions, refresh, remove } = useSessionStore()
  return { sessions, refresh, remove }
}

export function useShellNav() {
  const navigate = useNavigate()
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false)
  const { sessions, remove } = useSessionStore()
  const agentMgr = useAgentManager()
  const chat = useWorkspaceChat()

  useKeyboardShortcuts({
    onToggleSidebar: () => setSidebarCollapsed((v) => !v),
    onClosePanel: () => {},
  })

  const handleResumeSession = useCallback(
    async (id: string) => {
      try {
        const sess = await fetchSession(id)
        navigate(
          `/agents/${encodeURIComponent(sess.agent)}?resume=${encodeURIComponent(id)}`
        )
      } catch (err) {
        console.error("Failed to resume session:", err)
      }
    },
    [navigate]
  )

  const handleNewSession = useCallback(() => {
    chat.clear()
    navigate("/")
  }, [chat, navigate])

  return {
    navigate,
    sidebarCollapsed,
    setSidebarCollapsed,
    agentMgr,
    agents: agentMgr.agents,
    onToggleSidebar: () => setSidebarCollapsed((v) => !v),
    onOpenSettings: () => navigate("/settings"),
    onOpenManage: () => navigate("/manage"),
    onOpenKanban: () => navigate("/kanban"),
    onOpenApiDocs: () => navigate("/api-docs"),
    onOpenUsage: () => navigate("/usage"),
    onOpenRequestLogs: () => navigate("/request-logs"),
    onOpenAdmin: () => navigate("/admin"),
    onNewSession: handleNewSession,
    sessions,
    onResumeSession: handleResumeSession,
    onDeleteSession: remove,
  }
}

export type ShellNav = ReturnType<typeof useShellNav>

export function shellLayoutProps(
  shell: ShellNav,
  theme: string,
  setTheme: (t: "dark" | "light" | "system") => void
) {
  return {
    sidebarCollapsed: shell.sidebarCollapsed,
    onToggleSidebar: shell.onToggleSidebar,
    onOpenSettings: shell.onOpenSettings,
    onOpenManage: shell.onOpenManage,
    onOpenKanban: shell.onOpenKanban,
    onOpenApiDocs: shell.onOpenApiDocs,
    onOpenUsage: shell.onOpenUsage,
    onOpenRequestLogs: shell.onOpenRequestLogs,
    onOpenAdmin: shell.onOpenAdmin,
    onNewSession: shell.onNewSession,
    sessions: shell.sessions,
    agents: shell.agents,
    onResumeSession: shell.onResumeSession,
    onDeleteSession: shell.onDeleteSession,
    agent: {
      id: "",
      name: "nexa",
      provider: "",
      model: "",
      tools: [] as string[],
      maxTurns: 10,
    },
    status: "idle" as const,
    theme,
    onToggleTheme: () => setTheme(theme === "dark" ? "light" : "dark"),
  }
}
