import { useState } from "react"
import { Link, Navigate } from "react-router-dom"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import { $api } from "../api/client"
import Card from "../components/ui/Card"
import Input from "../components/ui/Input"
import Button from "../components/ui/Button"

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
  const [success, setSuccess] = useState(false)
  const { data: appConfig, isLoading: configLoading } = $api.useQuery(
    "get",
    "/app_config"
  )

  const mutation = $api.useMutation("post", "/register")

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<FormData>({ resolver: zodResolver(schema) })

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

  if (!configLoading && !appConfig?.registrationEnabled) {
    return <Navigate to="/login" replace />
  }

  if (success) {
    return (
      <div className="flex min-h-[calc(100vh-var(--nav-height))] items-center justify-center px-4">
        <Card className="w-full max-w-sm p-8 shadow-sm">
          <h1 className="mb-4 text-xl font-semibold text-text">
            Check your email
          </h1>
          <p className="text-sm text-text-secondary">
            We sent a confirmation link to your email. Please click it to
            activate your account.
          </p>
        </Card>
      </div>
    )
  }

  return (
    <div className="flex min-h-[calc(100vh-var(--nav-height))] items-center justify-center px-4">
      <Card className="w-full max-w-sm p-8 shadow-sm">
        <h1 className="mb-6 text-xl font-semibold text-text">
          Create an account
        </h1>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <Input
            label="Email"
            type="email"
            autoComplete="email"
            error={errors.email?.message}
            {...register("email")}
          />
          <Input
            label="Name"
            type="text"
            autoComplete="name"
            error={errors.name?.message}
            {...register("name")}
          />
          <Input
            label="Password"
            type="password"
            autoComplete="new-password"
            error={errors.password?.message}
            {...register("password")}
          />
          <Input
            label="Confirm password"
            type="password"
            autoComplete="new-password"
            error={errors.confirmPassword?.message}
            {...register("confirmPassword")}
          />
          {mutation.error && (
            <p role="alert" className="text-sm text-error">
              {mutation.error.message}
            </p>
          )}
          <Button
            type="submit"
            variant="primary"
            disabled={isSubmitting}
            className="w-full"
          >
            {isSubmitting ? "Creating account..." : "Create account"}
          </Button>
        </form>
        <p className="mt-4 text-center text-sm text-text-secondary">
          Already have an account?{" "}
          <Link to="/login" className="text-text hover:underline">
            Sign in
          </Link>
        </p>
      </Card>
    </div>
  )
}
