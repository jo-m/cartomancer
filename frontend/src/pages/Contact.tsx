import { useAppConfig } from "../api/client"
import useDocumentTitle from "../hooks/useDocumentTitle"
import PageContainer from "../components/ui/PageContainer"
import { EnvelopeIcon } from "@heroicons/react/24/outline"

/** Contact page with feedback email address fetched from app config. */
export default function Contact() {
  useDocumentTitle("Contact")
  const { data: appConfig, isLoading } = useAppConfig()

  if (isLoading || !appConfig?.contactEmail) {
    return null
  }

  return (
    <PageContainer size="md">
      <div className="rounded-lg border-2 border-primary/30 bg-primary/5 px-6 py-8 text-center">
        <EnvelopeIcon className="mx-auto mb-3 h-10 w-10 text-primary" />
        <p className="mb-1 text-sm uppercase tracking-wider text-text-muted">
          Feedback &amp; Contact
        </p>
        <a
          href={`mailto:${appConfig.contactEmail}`}
          className="text-2xl font-semibold text-primary hover:text-primary/80 transition-colors break-all"
        >
          {appConfig.contactEmail}
        </a>
        <p className="mt-3 text-sm text-text-secondary">
          Questions, suggestions, or bug reports - write anytime. A real human
          will read your message.
        </p>
      </div>
    </PageContainer>
  )
}
