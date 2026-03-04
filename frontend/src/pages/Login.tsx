import { useEffect } from "react"
import { Link, useNavigate } from "react-router-dom"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import { useSession } from "../context/SessionContext"

const schema = z.object({
  email: z.string().min(1, "Required").email("Invalid email"),
  password: z.string().min(1, "Required"),
})

type FormData = z.infer<typeof schema>

export default function Login() {
  const { user, loading, login } = useSession()
  const navigate = useNavigate()
  const {
    register,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = useForm<FormData>({ resolver: zodResolver(schema) })

  useEffect(() => {
    if (!loading && user) navigate("/")
  }, [user, loading, navigate])

  async function onSubmit(data: FormData) {
    try {
      await login(data.email, data.password)
      navigate("/")
    } catch (err) {
      setError("root", {
        message: err instanceof Error ? err.message : "Login failed",
      })
    }
  }

  return (
    <div className="flex min-h-[calc(100vh-57px)] items-center justify-center">
      <div className="w-full max-w-sm rounded-lg border border-gray-200 bg-white p-8 shadow-sm">
        <h1 className="mb-6 text-xl font-semibold text-gray-900">Sign in</h1>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div>
            <label className="mb-1 block text-sm font-medium text-gray-700">
              Email
            </label>
            <input
              type="email"
              autoComplete="email"
              {...register("email")}
              className="w-full rounded border border-gray-300 px-3 py-2 text-sm focus:border-gray-500 focus:outline-none"
            />
            {errors.email && (
              <p className="mt-1 text-sm text-red-600">
                {errors.email.message}
              </p>
            )}
          </div>
          <div>
            <label className="mb-1 block text-sm font-medium text-gray-700">
              Password
            </label>
            <input
              type="password"
              autoComplete="current-password"
              {...register("password")}
              className="w-full rounded border border-gray-300 px-3 py-2 text-sm focus:border-gray-500 focus:outline-none"
            />
            {errors.password && (
              <p className="mt-1 text-sm text-red-600">
                {errors.password.message}
              </p>
            )}
          </div>
          {errors.root && (
            <p className="text-sm text-red-600">{errors.root.message}</p>
          )}
          <button
            type="submit"
            disabled={isSubmitting}
            className="w-full cursor-pointer rounded bg-gray-900 px-4 py-2 text-sm font-medium text-white hover:bg-gray-700 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {isSubmitting ? "Signing in…" : "Sign in"}
          </button>
        </form>
        <p className="mt-4 text-center text-sm text-gray-600">
          Don&apos;t have an account?{" "}
          <Link to="/register" className="text-gray-900 hover:underline">
            Create one
          </Link>
        </p>
      </div>
    </div>
  )
}
