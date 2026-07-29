import { Link, useNavigate } from "react-router"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import { useSession } from "../context/SessionContext"
import { useAppConfig, ApiError } from "../api/client"
import useDocumentTitle from "../hooks/useDocumentTitle"
import useToast from "../hooks/useToast"
import Card from "../components/ui/Card"
import Input from "../components/ui/Input"
import Button from "../components/ui/Button"
import Alert from "../components/ui/Alert"
import Toast from "../components/Toast"

const schema = z.object({
  email: z.string().min(1, "Required").email("Invalid email"),
  password: z.string().min(1, "Required"),
})

type FormData = z.infer<typeof schema>

export default function Login() {
  useDocumentTitle("Login")
  const { login } = useSession()
  const { data: appConfig } = useAppConfig()
  const navigate = useNavigate()
  const { toast, showToast, dismissToast } = useToast()
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<FormData>({ resolver: zodResolver(schema) })

  async function onSubmit(data: FormData) {
    try {
      await login(data.email, data.password)
      navigate("/")
    } catch (err) {
      const message =
        err instanceof ApiError && err.status === 429
          ? "Too many login attempts. Please try again later."
          : err instanceof Error
            ? err.message
            : "Login failed"
      showToast(message)
    }
  }

  return (
    <div className="flex min-h-[calc(100vh-var(--nav-height))] items-center justify-center px-4">
      {toast && (
        <Toast
          key={toast.key}
          message={toast.message}
          variant={toast.variant}
          onDismiss={dismissToast}
        />
      )}
      <Card className="w-full max-w-sm p-8 shadow-sm">
        <h1 className="mb-6 text-xl font-semibold text-text">Sign in</h1>
        {appConfig?.demoMode && appConfig.demoEmail && (
          <Alert variant="info" className="mb-4">
            <p className="font-medium">Demo instance</p>
            <p>Data is periodically deleted. Users cannot be modified.</p>
            <br />
            <p>
              Email: <code className="font-mono">{appConfig.demoEmail}</code>
            </p>
            <p>
              Password:{" "}
              <code className="font-mono">{appConfig.demoPassword}</code>
            </p>
          </Alert>
        )}
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <Input
            label="Email"
            type="email"
            autoComplete="email"
            error={errors.email?.message}
            {...register("email")}
          />
          <Input
            label="Password"
            type="password"
            autoComplete="current-password"
            error={errors.password?.message}
            {...register("password")}
          />
          <Button
            type="submit"
            variant="primary"
            disabled={isSubmitting}
            className="w-full"
          >
            {isSubmitting ? "Signing in..." : "Sign in"}
          </Button>
        </form>
        {appConfig?.registrationEnabled && (
          <p className="mt-4 text-center text-sm text-text-secondary">
            Don&apos;t have an account?{" "}
            <Link to="/register" className="text-text hover:underline">
              Create one
            </Link>
          </p>
        )}
      </Card>
    </div>
  )
}
