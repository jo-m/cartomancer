import { Link } from "react-router-dom"
import Button from "../components/ui/Button"

export default function NotFound() {
  return (
    <div className="flex min-h-[calc(100vh-57px)] items-center justify-center">
      <div className="text-center">
        <h1 className="text-6xl font-bold text-text">404</h1>
        <p className="mt-4 text-lg text-text-secondary">
          The page you are looking for does not exist.
        </p>
        <Link to="/" className="mt-6 inline-block">
          <Button variant="primary">Back to home</Button>
        </Link>
      </div>
    </div>
  )
}
