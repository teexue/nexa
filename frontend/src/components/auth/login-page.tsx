import { Navigate } from "react-router"
import { useTranslation } from "react-i18next"
import { useAuth } from "@/lib/auth"
import { LoginScreen } from "./login-form"

/** Full-page login / register gate — shown before any app data loads. */
export function LoginPage({
  hasUsers: initialHasUsers,
}: {
  hasUsers?: boolean
}) {
  const { t } = useTranslation()
  const {
    state,
    hasUsers = initialHasUsers ?? false,
    allowRegistration,
    refresh,
  } = useAuth()

  if (state === "loading") {
    return (
      <div className="flex h-full min-h-svh items-center justify-center">
        <p className="text-sm text-muted-foreground">{t("common.loading")}</p>
      </div>
    )
  }

  if (state === "authenticated") {
    return <Navigate to="/" replace />
  }

  return (
    <LoginScreen
      hasUsers={hasUsers}
      allowRegistration={allowRegistration}
      refresh={refresh}
    />
  )
}
