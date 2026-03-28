import { Link } from "react-router-dom"

export default function NotFound() {
  return (
    <div className="flex min-h-[calc(100vh-57px)] items-center justify-center">
      <div className="text-center">
        <h1 className="text-6xl font-bold text-gray-900">404</h1>
        <p className="mt-4 text-lg text-gray-600">
          The page you are looking for does not exist.
        </p>
        <Link
          to="/"
          className="mt-6 inline-block rounded bg-gray-900 px-4 py-2 text-sm text-white hover:bg-gray-700"
        >
          Back to home
        </Link>
      </div>
    </div>
  )
}
