import { Link } from "react-router-dom"
import { useAppConfig } from "../api/client"
import { useSession } from "../context/SessionContext"
import PageContainer from "../components/ui/PageContainer"
import Button from "../components/ui/Button"

/** Welcome landing page shown to all visitors at the root route. */
export default function Welcome() {
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
        <Link to="/tracks">
          <Button variant="primary">Browse tracks</Button>
        </Link>
        {!user && appConfig?.registrationEnabled && (
          <Link to="/register">
            <Button variant="secondary">Create account</Button>
          </Link>
        )}
      </div>
    </PageContainer>
  )
}
