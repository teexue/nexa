import { useState } from "react"
import { useNavigate } from "react-router"
import { useTranslation } from "react-i18next"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { loginUser, registerUser, setAccessToken } from "@/lib/api"
import { isComposingEvent } from "@/lib/keys"

type Mode = "login" | "register"

interface LoginFormModel {
  effectiveMode: Mode
  canRegister: boolean
  username: string
  setUsername: (value: string) => void
  password: string
  setPassword: (value: string) => void
  displayName: string
  setDisplayName: (value: string) => void
  saving: boolean
  error: string | null
  handleSubmit: () => Promise<void>
  toggleMode: () => void
}

async function submitAuth(opts: {
  mode: Mode
  username: string
  password: string
  displayName: string
  refresh: () => Promise<void>
  navigate: ReturnType<typeof useNavigate>
}) {
  const session =
    opts.mode === "register"
      ? await registerUser(opts.username, opts.password, opts.displayName)
      : await loginUser(opts.username, opts.password)
  setAccessToken(session.token)
  await opts.refresh()
  opts.navigate("/", { replace: true })
}

function useLoginForm(opts: {
  hasUsers: boolean
  allowRegistration: boolean
  refresh: () => Promise<void>
}): LoginFormModel {
  const navigate = useNavigate()
  const [mode, setMode] = useState<Mode>(opts.hasUsers ? "login" : "register")
  const [username, setUsername] = useState("")
  const [password, setPassword] = useState("")
  const [displayName, setDisplayName] = useState("")
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const canRegister = !opts.hasUsers || opts.allowRegistration
  const effectiveMode: Mode =
    mode === "register" && !canRegister ? "login" : mode

  const handleSubmit = async () => {
    setSaving(true)
    setError(null)
    try {
      await submitAuth({
        mode: effectiveMode,
        username,
        password,
        displayName,
        refresh: opts.refresh,
        navigate,
      })
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setSaving(false)
    }
  }

  const toggleMode = () => {
    setMode(effectiveMode === "login" ? "register" : "login")
    setError(null)
  }

  return {
    effectiveMode,
    canRegister,
    username,
    setUsername,
    password,
    setPassword,
    displayName,
    setDisplayName,
    saving,
    error,
    handleSubmit,
    toggleMode,
  }
}

function LoginBrand({ mode }: { mode: Mode }) {
  const { t } = useTranslation()
  return (
    <div className="flex flex-col items-center gap-3 text-center">
      <img src="/logo.png" alt="Nexa" className="h-12 w-12 rounded-xl" />
      <div>
        <h1 className="font-heading text-2xl tracking-tight text-foreground">
          Nexa
        </h1>
        <p className="mt-1 text-xs text-muted-foreground">
          {mode === "login" ? t("auth.loginHint") : t("auth.registerHint")}
        </p>
      </div>
    </div>
  )
}

function AuthField({
  label,
  children,
}: {
  label: string
  children: React.ReactNode
}) {
  return (
    <div className="space-y-1.5">
      <Label className="text-xs text-muted-foreground">{label}</Label>
      {children}
    </div>
  )
}

function LoginFields({ form }: { form: LoginFormModel }) {
  const { t } = useTranslation()
  const submitOnEnter = (e: React.KeyboardEvent) => {
    if (isComposingEvent(e)) return
    if (e.key === "Enter") void form.handleSubmit()
  }
  return (
    <>
      <AuthField label={t("auth.username")}>
        <Input
          value={form.username}
          onChange={(e) => form.setUsername(e.target.value)}
          className="h-10 rounded-xl font-mono text-sm"
          autoComplete="username"
          autoFocus
          onKeyDown={submitOnEnter}
        />
      </AuthField>
      <AuthField label={t("auth.password")}>
        <Input
          type="password"
          value={form.password}
          onChange={(e) => form.setPassword(e.target.value)}
          className="h-10 rounded-xl text-sm"
          autoComplete={
            form.effectiveMode === "login" ? "current-password" : "new-password"
          }
          onKeyDown={submitOnEnter}
        />
      </AuthField>
      {form.effectiveMode === "register" && (
        <AuthField label={t("auth.displayName")}>
          <Input
            value={form.displayName}
            onChange={(e) => form.setDisplayName(e.target.value)}
            className="h-10 rounded-xl text-sm"
            autoComplete="nickname"
          />
        </AuthField>
      )}
    </>
  )
}

function LoginActions({ form }: { form: LoginFormModel }) {
  const { t } = useTranslation()
  const disabled =
    form.saving ||
    !form.username.trim() ||
    (form.effectiveMode === "register"
      ? form.password.length < 6
      : form.password.length === 0)
  return (
    <>
      {form.error && <p className="text-xs text-destructive">{form.error}</p>}
      <Button
        className="h-10 w-full text-sm"
        onClick={() => void form.handleSubmit()}
        disabled={disabled}
      >
        {form.saving
          ? t("common.loading")
          : form.effectiveMode === "login"
            ? t("auth.login")
            : t("auth.register")}
      </Button>
      {(form.canRegister || form.effectiveMode === "register") && (
        <button
          type="button"
          className="w-full text-center text-xs text-muted-foreground hover:text-foreground"
          onClick={form.toggleMode}
        >
          {form.effectiveMode === "login"
            ? t("auth.switchRegister")
            : t("auth.switchLogin")}
        </button>
      )}
    </>
  )
}

function LoginCard({
  hasUsers,
  form,
}: {
  hasUsers: boolean
  form: LoginFormModel
}) {
  const { t } = useTranslation()
  return (
    <div className="glass-tile space-y-3 rounded-2xl border border-[color:var(--glass-edge)] p-5">
      {form.effectiveMode === "register" && !hasUsers && (
        <p className="rounded-lg bg-primary/10 px-3 py-2 text-[11px] leading-relaxed text-primary">
          {t("auth.firstUserAdminHint")}
        </p>
      )}
      <LoginFields form={form} />
      <LoginActions form={form} />
    </div>
  )
}

export function LoginScreen({
  hasUsers,
  allowRegistration,
  refresh,
}: {
  hasUsers: boolean
  allowRegistration: boolean
  refresh: () => Promise<void>
}) {
  const form = useLoginForm({ hasUsers, allowRegistration, refresh })
  return (
    <div className="flex h-full min-h-svh items-center justify-center px-4">
      <div className="w-full max-w-sm space-y-6">
        <LoginBrand mode={form.effectiveMode} />
        <LoginCard hasUsers={hasUsers} form={form} />
      </div>
    </div>
  )
}
