import { useEffect, useState } from "react"
import { useNavigate } from "react-router-dom"
import { useSession } from "../context/SessionContext"

export default function Login() {
  const { user, loading, login } = useSession()
  const navigate = useNavigate()
  const [email, setEmail] = useState("")
  const [password, setPassword] = useState("")
  const [emailError, setEmailError] = useState<string | null>(null)
  const [passwordError, setPasswordError] = useState<string | null>(null)
  const [submitError, setSubmitError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    if (!loading && user) navigate("/")
  }, [user, loading, navigate])

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setSubmitError(null)
    setSubmitting(true)
    try {
      await login(email, password)
      navigate("/")
    } catch (err) {
      setSubmitError(err instanceof Error ? err.message : "Login failed")
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="flex min-h-[calc(100vh-57px)] items-center justify-center">
      <div className="w-full max-w-sm rounded-lg border border-gray-200 bg-white p-8 shadow-sm">
        <h1 className="mb-6 text-xl font-semibold text-gray-900">Sign in</h1>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="mb-1 block text-sm font-medium text-gray-700">
              Email
            </label>
            <input
              type="email"
              value={email}
              required
              autoComplete="email"
              onChange={(e) => {
                setEmail(e.target.value)
                setEmailError(null)
              }}
              onInvalid={(e) => {
                e.preventDefault()
                setEmailError(e.currentTarget.validationMessage)
              }}
              className="w-full rounded border border-gray-300 px-3 py-2 text-sm focus:border-gray-500 focus:outline-none"
            />
            {emailError && (
              <p className="mt-1 text-sm text-red-600">{emailError}</p>
            )}
          </div>
          <div>
            <label className="mb-1 block text-sm font-medium text-gray-700">
              Password
            </label>
            <input
              type="password"
              value={password}
              required
              autoComplete="current-password"
              onChange={(e) => {
                setPassword(e.target.value)
                setPasswordError(null)
              }}
              onInvalid={(e) => {
                e.preventDefault()
                setPasswordError(e.currentTarget.validationMessage)
              }}
              className="w-full rounded border border-gray-300 px-3 py-2 text-sm focus:border-gray-500 focus:outline-none"
            />
            {passwordError && (
              <p className="mt-1 text-sm text-red-600">{passwordError}</p>
            )}
          </div>
          {submitError && <p className="text-sm text-red-600">{submitError}</p>}
          <button
            type="submit"
            disabled={submitting}
            className="w-full cursor-pointer rounded bg-gray-900 px-4 py-2 text-sm font-medium text-white hover:bg-gray-700 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {submitting ? "Signing in…" : "Sign in"}
          </button>
        </form>
      </div>
    </div>
  )
}
