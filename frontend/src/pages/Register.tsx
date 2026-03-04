import { useEffect, useState } from "react"
import { Link, useNavigate } from "react-router-dom"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import { useSession } from "../context/SessionContext"
import { $api } from "../api/client"

const schema = z
  .object({
    email: z.string().min(1, "Required").email("Invalid email"),
    name: z.string().min(1, "Required"),
    password: z.string().min(1, "Required"),
    confirmPassword: z.string().min(1, "Required"),
  })
  .refine((d) => d.password === d.confirmPassword, {
    message: "Passwords do not match",
    path: ["confirmPassword"],
  })

type FormData = z.infer<typeof schema>

export default function Register() {
  const { user, loading } = useSession()
  const navigate = useNavigate()
  const [success, setSuccess] = useState(false)

  const mutation = $api.useMutation("post", "/register")

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<FormData>({ resolver: zodResolver(schema) })

  useEffect(() => {
    if (!loading && user) navigate("/")
  }, [user, loading, navigate])

  async function onSubmit(data: FormData) {
    try {
      await mutation.mutateAsync({
        body: { email: data.email, name: data.name, password: data.password },
      })
      setSuccess(true)
    } catch {
      // error displayed via mutation.error
    }
  }

  if (success) {
    return (
      <div className="flex min-h-[calc(100vh-57px)] items-center justify-center">
        <div className="w-full max-w-sm rounded-lg border border-gray-200 bg-white p-8 shadow-sm">
          <h1 className="mb-4 text-xl font-semibold text-gray-900">
            Check your email
          </h1>
          <p className="text-sm text-gray-600">
            We sent a confirmation link to your email. Please click it to
            activate your account.
          </p>
        </div>
      </div>
    )
  }

  return (
    <div className="flex min-h-[calc(100vh-57px)] items-center justify-center">
      <div className="w-full max-w-sm rounded-lg border border-gray-200 bg-white p-8 shadow-sm">
        <h1 className="mb-6 text-xl font-semibold text-gray-900">
          Create an account
        </h1>
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
              Name
            </label>
            <input
              type="text"
              autoComplete="name"
              {...register("name")}
              className="w-full rounded border border-gray-300 px-3 py-2 text-sm focus:border-gray-500 focus:outline-none"
            />
            {errors.name && (
              <p className="mt-1 text-sm text-red-600">{errors.name.message}</p>
            )}
          </div>
          <div>
            <label className="mb-1 block text-sm font-medium text-gray-700">
              Password
            </label>
            <input
              type="password"
              autoComplete="new-password"
              {...register("password")}
              className="w-full rounded border border-gray-300 px-3 py-2 text-sm focus:border-gray-500 focus:outline-none"
            />
            {errors.password && (
              <p className="mt-1 text-sm text-red-600">
                {errors.password.message}
              </p>
            )}
          </div>
          <div>
            <label className="mb-1 block text-sm font-medium text-gray-700">
              Confirm password
            </label>
            <input
              type="password"
              autoComplete="new-password"
              {...register("confirmPassword")}
              className="w-full rounded border border-gray-300 px-3 py-2 text-sm focus:border-gray-500 focus:outline-none"
            />
            {errors.confirmPassword && (
              <p className="mt-1 text-sm text-red-600">
                {errors.confirmPassword.message}
              </p>
            )}
          </div>
          {mutation.error && (
            <p className="text-sm text-red-600">
              {(mutation.error as unknown as Error).message}
            </p>
          )}
          <button
            type="submit"
            disabled={isSubmitting}
            className="w-full cursor-pointer rounded bg-gray-900 px-4 py-2 text-sm font-medium text-white hover:bg-gray-700 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {isSubmitting ? "Creating account…" : "Create account"}
          </button>
        </form>
        <p className="mt-4 text-center text-sm text-gray-600">
          Already have an account?{" "}
          <Link to="/login" className="text-gray-900 hover:underline">
            Sign in
          </Link>
        </p>
      </div>
    </div>
  )
}
