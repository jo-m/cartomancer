import useDocumentTitle from "../hooks/useDocumentTitle"
import Button from "../components/ui/Button"

export default function Forbidden() {
  useDocumentTitle("Forbidden")
  return (
    <div className="flex min-h-[calc(100vh-var(--nav-height))] items-center justify-center">
      <div className="text-center" role="alert">
        <h1 className="text-6xl font-bold text-text">403</h1>
        <p className="mt-4 text-lg text-text-secondary">
          You do not have permission to access this page.
        </p>
        <Button to="/" variant="primary" className="mt-6">
          Back to home
        </Button>
      </div>
    </div>
  )
}
