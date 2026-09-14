import { Trans, useTranslation } from "react-i18next"
import { Plus } from "lucide-react"
import { Button } from "@/components/ui/button"

export function WorkspaceEmptyState({
  noAgent,
  onCreateAgent,
}: {
  noAgent?: boolean
  onCreateAgent?: () => void
}) {
  const { t } = useTranslation()
  return (
    <div className="flex h-full items-center justify-center px-6">
      <div className="max-w-sm text-center">
        <div className="relative mx-auto mb-5 h-14 w-14">
          <img
            src="/logo.png"
            alt="Nexa logo"
            className="h-full w-full object-contain"
          />
        </div>
        {noAgent ? (
          <NoAgentCopy onCreateAgent={onCreateAgent} />
        ) : (
          <>
            <p className="font-heading text-base text-foreground">
              {t("conversation.workspaceTitle")}
            </p>
            <p className="mt-2 text-xs leading-relaxed text-muted-foreground">
              {t("conversation.workspaceDesc")}
            </p>
          </>
        )}
      </div>
    </div>
  )
}

function NoAgentCopy({ onCreateAgent }: { onCreateAgent?: () => void }) {
  const { t } = useTranslation()
  return (
    <>
      <p className="font-heading text-base text-foreground">
        {t("conversation.noAgentTitle")}
      </p>
      <p className="mt-2 text-xs leading-relaxed text-muted-foreground">
        <Trans
          i18nKey="conversation.noAgentDesc"
          components={{
            code: <code className="rounded bg-muted px-1 py-0.5 font-mono" />,
          }}
        />
      </p>
      {onCreateAgent && (
        <Button
          variant="outline"
          size="sm"
          className="mt-4 gap-1.5 text-xs"
          onClick={onCreateAgent}
        >
          <Plus className="h-3.5 w-3.5" /> {t("common.createAgent")}
        </Button>
      )}
    </>
  )
}
