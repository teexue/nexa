import {
  createContext,
  useCallback,
  useContext,
  useRef,
  useState,
  type ReactNode,
} from "react"
import { CheckCircle2, AlertTriangle, XCircle, Info, X } from "lucide-react"

type ToastVariant = "success" | "error" | "warning" | "info"

interface ToastItem {
  id: number
  variant: ToastVariant
  title: string
  description?: string
  duration: number
}

interface ToastInput {
  title: string
  description?: string
  duration?: number
}

interface ToastApi {
  success: (t: ToastInput | string) => void
  error: (t: ToastInput | string) => void
  warning: (t: ToastInput | string) => void
  info: (t: ToastInput | string) => void
  dismiss: (id: number) => void
}

const ToastContext = createContext<ToastApi | null>(null)

const VARIANT_STYLE: Record<ToastVariant, string> = {
  success: "border-success/40 bg-success/10",
  error: "border-destructive/40 bg-destructive/10",
  warning: "border-warning/40 bg-warning/10",
  info: "border-primary/40 bg-primary/10",
}

const VARIANT_ICON: Record<ToastVariant, ReactNode> = {
  success: <CheckCircle2 className="h-4 w-4 text-success" />,
  error: <XCircle className="h-4 w-4 text-destructive" />,
  warning: <AlertTriangle className="h-4 w-4 text-warning" />,
  info: <Info className="h-4 w-4 text-primary" />,
}

function normalize(input: ToastInput | string): ToastInput {
  return typeof input === "string" ? { title: input } : input
}

/** Provides global toast notifications. Mount once near the app root. */
export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<ToastItem[]>([])
  const seqRef = useRef(0)

  const dismiss = useCallback((id: number) => {
    setToasts((prev) => prev.filter((t) => t.id !== id))
  }, [])

  const push = useCallback(
    (variant: ToastVariant, input: ToastInput | string) => {
      const { title, description, duration = 4000 } = normalize(input)
      const id = ++seqRef.current
      setToasts((prev) => [
        ...prev,
        { id, variant, title, description, duration },
      ])
      if (duration > 0) {
        window.setTimeout(() => dismiss(id), duration)
      }
    },
    [dismiss]
  )

  const api: ToastApi = {
    success: (t) => push("success", t),
    error: (t) => push("error", t),
    warning: (t) => push("warning", t),
    info: (t) => push("info", t),
    dismiss,
  }

  return (
    <ToastContext.Provider value={api}>
      {children}
      <Toaster toasts={toasts} onDismiss={dismiss} />
    </ToastContext.Provider>
  )
}

function Toaster({
  toasts,
  onDismiss,
}: {
  toasts: ToastItem[]
  onDismiss: (id: number) => void
}) {
  return (
    <div className="pointer-events-none fixed right-4 bottom-4 z-[100] flex w-full max-w-sm flex-col gap-2">
      {toasts.map((t) => (
        <ToastCard key={t.id} toast={t} onDismiss={onDismiss} />
      ))}
    </div>
  )
}

function ToastCard({
  toast,
  onDismiss,
}: {
  toast: ToastItem
  onDismiss: (id: number) => void
}) {
  return (
    <div
      className={`glass-panel pointer-events-auto flex items-start gap-3 rounded-2xl border border-[color:var(--glass-edge)] px-4 py-3 ${VARIANT_STYLE[toast.variant]}`}
      role="status"
    >
      <span className="mt-0.5 shrink-0">{VARIANT_ICON[toast.variant]}</span>
      <div className="min-w-0 flex-1">
        <p className="text-sm leading-5 font-medium">{toast.title}</p>
        {toast.description && (
          <p className="mt-0.5 text-xs leading-4 text-muted-foreground">
            {toast.description}
          </p>
        )}
      </div>
      <button
        type="button"
        onClick={() => onDismiss(toast.id)}
        className="shrink-0 rounded p-0.5 text-muted-foreground hover:text-foreground"
        aria-label="close"
      >
        <X className="h-3.5 w-3.5" />
      </button>
    </div>
  )
}

/** Access the toast API. Must be used within a ToastProvider. */
export function useToast(): ToastApi {
  const ctx = useContext(ToastContext)
  if (!ctx) throw new Error("useToast must be used within ToastProvider")
  return ctx
}
