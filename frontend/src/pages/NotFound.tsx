import useDocumentTitle from "../hooks/useDocumentTitle"
import Button from "../components/ui/Button"

export default function NotFound() {
  useDocumentTitle("Not Found")
  return (
    <div className="flex min-h-[calc(100vh-var(--nav-height))] items-center justify-center">
      <div className="text-center">
        <h1 className="text-6xl font-bold text-text">404</h1>
        <p className="mt-4 text-lg text-text-secondary">
          The page you are looking for does not exist.
        </p>
        <Button to="/" variant="primary" className="mt-6">
          Back to home
        </Button>
      </div>
    </div>
  )
}
