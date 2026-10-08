import {
  useCallback,
  useRef,
  type Dispatch,
  type MutableRefObject,
} from "react"
import { abortSession, fetchSession } from "@/lib/api"
import { buildSendPrompt, imagesFromAttachments } from "@/lib/attachments"
import type { FileAttachment } from "@/types/agent"
import type { ChatAction } from "./use-chat-state"
import { fromBackendMessages } from "./use-chat-messages"
import type { BackendMsg } from "./use-chat-messages"
import { sendRunRequest } from "./chat-run"
import { followLiveRun, type ResumeMeta } from "./chat-follow"
import { writeLastSessionId } from "./last-session"

type AbortRef = MutableRefObject<AbortController | null>
type SessionRef = MutableRefObject<string | null>
type GenRef = MutableRefObject<number>

export interface SendMessageOpts {
  text: string
  agent: string
  workDir?: string
  attachments?: FileAttachment[]
  model?: string
  provider?: string
  thinkingEffort?: string
}

function detachLocal(abortRef: AbortRef): void {
  abortRef.current?.abort()
  abortRef.current = null
}

function closeStream(dispatch: Dispatch<ChatAction>): void {
  dispatch({
    type: "STREAM_DONE",
    entryId: "",
    status: "cancelled",
    turns: 0,
  })
}

function cancelRun(
  dispatch: Dispatch<ChatAction>,
  abortRef: AbortRef,
  sessionId: string | null
): void {
  detachLocal(abortRef)
  if (sessionId) void abortSession(sessionId).catch(() => {})
  closeStream(dispatch)
}

export function useChatSend(
  dispatch: Dispatch<ChatAction>,
  abortRef: AbortRef,
  sessionIdRef: SessionRef
) {
  return useCallback(
    async (opts: SendMessageOpts) => {
      detachLocal(abortRef)
      closeStream(dispatch)
      const controller = new AbortController()
      abortRef.current = controller
      dispatch({
        type: "ADD_USER_MESSAGE",
        text: opts.text,
        attachments: opts.attachments,
      })
      const entryId = `assistant-${Date.now()}`
      dispatch({ type: "START_ASSISTANT", entryId })
      try {
        await sendRunRequest({
          agent: opts.agent,
          prompt: buildSendPrompt(opts.text, opts.attachments ?? []),
          sessionId: sessionIdRef.current,
          workDir: opts.workDir,
          signal: controller.signal,
          entryId,
          dispatch,
          images: imagesFromAttachments(opts.attachments),
          model: opts.model,
          provider: opts.provider,
          thinkingEffort: opts.thinkingEffort,
        })
      } catch (err: unknown) {
        if (err instanceof DOMException && err.name === "AbortError") return
        dispatch({
          type: "STREAM_ERROR",
          message: err instanceof Error ? err.message : "Unknown error",
        })
      }
    },
    [dispatch, abortRef, sessionIdRef]
  )
}

export function useChatSessionOps(
  dispatch: Dispatch<ChatAction>,
  abortRef: AbortRef,
  sessionIdRef: SessionRef
) {
  const genRef = useRef(0)
  const abort = useCallback(() => {
    cancelRun(dispatch, abortRef, sessionIdRef.current)
  }, [dispatch, abortRef, sessionIdRef])
  const clear = useCallback(() => {
    abort()
    writeLastSessionId(null)
    dispatch({ type: "CLEAR" })
  }, [abort, dispatch])
  const loadSession = useCallback(
    (
      sessionId: string,
      messages: BackendMsg[],
      metadata?: Record<string, string>
    ) => {
      detachLocal(abortRef)
      dispatch({
        type: "LOAD_SESSION",
        sessionId,
        messages: fromBackendMessages(messages),
        metadata,
      })
    },
    [abortRef, dispatch]
  )
  const resumeSession = useResumeSession(
    dispatch,
    abortRef,
    sessionIdRef,
    genRef,
    loadSession
  )
  const setSessionId = useCallback(
    (sessionId: string | null) => {
      dispatch({ type: "SET_SESSION_ID", sessionId })
    },
    [dispatch]
  )
  return { abort, clear, loadSession, resumeSession, setSessionId }
}

function useResumeSession(
  dispatch: Dispatch<ChatAction>,
  abortRef: AbortRef,
  sessionIdRef: SessionRef,
  genRef: GenRef,
  loadSession: (
    sessionId: string,
    messages: BackendMsg[],
    metadata?: Record<string, string>
  ) => void
) {
  return useCallback(
    async (sessionId: string): Promise<ResumeMeta | null> => {
      if (isAttached(sessionIdRef, abortRef, sessionId)) {
        return fetchResumeMeta(sessionId)
      }
      const token = ++genRef.current
      detachLocal(abortRef)
      const controller = new AbortController()
      abortRef.current = controller
      try {
        return await resumeOrLoad({
          sessionId,
          signal: controller.signal,
          dispatch,
          token,
          genRef,
          loadSession,
        })
      } catch (err) {
        if (err instanceof DOMException && err.name === "AbortError")
          return null
        console.error("Failed to resume session:", err)
        return null
      }
    },
    [dispatch, abortRef, sessionIdRef, genRef, loadSession]
  )
}

function isAttached(
  sessionIdRef: SessionRef,
  abortRef: AbortRef,
  sessionId: string
): boolean {
  return (
    sessionIdRef.current === sessionId &&
    !!abortRef.current &&
    !abortRef.current.signal.aborted
  )
}

async function fetchResumeMeta(sessionId: string): Promise<ResumeMeta | null> {
  try {
    const sess = await fetchSession(sessionId)
    return {
      agent: sess.agent,
      workdir: sess.metadata?.workdir || null,
      model: sess.metadata?.model || "",
      provider: sess.metadata?.provider || "",
    }
  } catch (err) {
    console.error("Failed to resume session:", err)
    return null
  }
}

interface ResumeLoadOpts {
  sessionId: string
  signal: AbortSignal
  dispatch: Dispatch<ChatAction>
  token: number
  genRef: GenRef
  loadSession: (
    sessionId: string,
    messages: BackendMsg[],
    metadata?: Record<string, string>
  ) => void
}

async function resumeOrLoad(opts: ResumeLoadOpts): Promise<ResumeMeta | null> {
  const live = await followLiveRun(opts.sessionId, opts.signal, opts.dispatch)
  if (opts.token !== opts.genRef.current) return null
  if (live) return live
  const sess = await fetchSession(opts.sessionId)
  if (opts.token !== opts.genRef.current || opts.signal.aborted) return null
  opts.loadSession(opts.sessionId, sess.messages as BackendMsg[], sess.metadata)
  return {
    agent: sess.agent,
    workdir: sess.metadata?.workdir || null,
    model: sess.metadata?.model || "",
    provider: sess.metadata?.provider || "",
  }
}
