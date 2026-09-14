import { useEffect } from "react"
import { createPortal } from "react-dom"
import { WorkspacePanel } from "@/components/conversation/workspace-panel"
import { ConversationActions } from "@/components/conversation/conversation-actions"
import { SessionWorkdir } from "@/components/conversation/session-workdir"
import { ModelPicker } from "@/components/conversation/model-picker"
import { AppDialogs } from "./app-dialogs"
import { setWorkspaceChrome, clearWorkspaceChrome } from "./workspace-chrome"
import type { useWorkspacePage } from "./use-workspace-page"

type WorkspacePage = ReturnType<typeof useWorkspacePage>

function workspaceChrome(page: WorkspacePage) {
  const { agent, derived, actions, chat } = page
  return {
    agent: agent.agentInfo ?? {
      id: "",
      name: "nexa",
      provider: "",
      model: "",
      tools: [],
      maxTurns: 10,
    },
    agentLocked: derived.agentLocked,
    onSelectAgent: actions.handleSelectAgent,
    onNewSession: actions.handleNewSession,
    activeSessionId: chat.sessionId,
    status: derived.status,
  }
}

function WorkspaceChat({ page }: { page: WorkspacePage }) {
  const { chat, search, ui, agent, derived, actions, navigate } = page
  return (
    <WorkspacePanel
      messages={chat.messages}
      isStreaming={chat.isStreaming}
      error={chat.error}
      onSendMessage={actions.handleSendMessage}
      onStop={chat.abort}
      selectedToolCallId={ui.selectedToolCallId}
      onSelectToolCall={actions.handleSelectToolCall}
      onApproveTool={(id) => actions.resolveToolApproval(id, true)}
      onDenyTool={(id) => actions.resolveToolApproval(id, false)}
      noAgent={!derived.hasAgents}
      onCreateAgent={() => navigate("/manage/agents/new")}
      agentName={agent.agentInfo?.name || agent.resolvedAgent}
      visionEnabled={derived.visionEnabled}
      search={search}
      inputAccessory={
        <>
          <ModelPicker
            providers={ui.providers}
            provider={ui.runModel.provider || agent.agentInfo?.provider || ""}
            model={ui.runModel.model || agent.agentInfo?.model || ""}
            locked={derived.agentLocked}
            onChange={actions.setRunModel}
          />
          <SessionWorkdir
            workDir={ui.workDir}
            sessionScoped={ui.sessionWorkDir !== null}
            onPick={(dir) => void actions.handleWorkdirChange(dir)}
            onClear={() => void actions.handleWorkdirChange("")}
          />
        </>
      }
      tokenUsage={derived.tokenUsage}
      sessionId={chat.sessionId}
    />
  )
}

function WorkspaceTopbarActions({ page }: { page: WorkspacePage }) {
  const host = document.getElementById("shell-topbar-actions")
  const { search, chat, agent } = page
  if (!host) return null
  return createPortal(
    <ConversationActions
      searchOpen={search.searchOpen}
      onToggleSearch={search.toggleOpen}
      messages={chat.messages}
      agentName={agent.agentInfo?.name || agent.resolvedAgent}
    />,
    host
  )
}

export function WorkspaceShell({ page }: { page: WorkspacePage }) {
  useEffect(() => {
    setWorkspaceChrome(workspaceChrome(page))
  }, [page])
  useEffect(() => () => clearWorkspaceChrome(), [])
  return (
    <>
      <WorkspaceTopbarActions page={page} />
      <WorkspaceChat page={page} />
      <AppDialogs
        agentMgr={page.agentMgr}
        selectedTool={null}
        setSelectedTool={() => {}}
      />
    </>
  )
}
