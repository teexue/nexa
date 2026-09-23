import { Navigate, useLocation } from "react-router"
import { useTranslation } from "react-i18next"
import { useAuth } from "@/lib/auth"

/** Blocks app routes until a valid JWT session exists — no data hooks run above this. */
export function RequireAuth({ children }: { children: React.ReactNode }) {
  const { t } = useTranslation()
  const { state } = useAuth()
  const location = useLocation()

  if (state === "loading") {
    return (
      <div className="flex h-full min-h-svh items-center justify-center">
        <p className="text-sm text-muted-foreground">{t("common.loading")}</p>
      </div>
    )
  }

  if (state !== "authenticated") {
    return <Navigate to="/login" replace state={{ from: location.pathname }} />
  }

  return <>{children}</>
}
