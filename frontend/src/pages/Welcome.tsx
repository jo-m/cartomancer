import { useAppConfig } from "../api/client"
import { useSession } from "../context/SessionContext"
import useDocumentTitle from "../hooks/useDocumentTitle"
import PageContainer from "../components/ui/PageContainer"
import Button from "../components/ui/Button"

/** Welcome landing page shown to all visitors at the root route. */
export default function Welcome() {
  useDocumentTitle("")
  const { data: appConfig } = useAppConfig()
  const { user } = useSession()

  return (
    <PageContainer className="py-16">
      <h1 className="mb-4 text-3xl font-semibold tracking-wide text-text">
        {appConfig?.instanceName}
      </h1>
      <p className="mb-8 text-text-secondary">
        The gpx track library with a touch of magic.
      </p>
      <div className="flex gap-4">
        {user ? (
          <>
            <Button to="/tracks" variant="primary">
              Public tracks
            </Button>
            <Button to="/account/tracks" variant="secondary">
              My tracks
            </Button>
          </>
        ) : (
          <>
            <Button to="/tracks" variant="primary">
              Browse tracks
            </Button>
            <Button to="/login" variant="secondary">
              Log in
            </Button>
            {appConfig?.registrationEnabled && (
              <Button to="/register" variant="secondary">
                Create account
              </Button>
            )}
          </>
        )}
      </div>
    </PageContainer>
  )
}
