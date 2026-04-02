import { useMemo, useState } from "react"
import { Link } from "react-router-dom"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import { $api, fetchClient } from "../api/client"
import { useSession } from "../context/SessionContext"
import { useUrlState, stringParam } from "../hooks/useUrlState"
import Toast from "../components/Toast"
import PageContainer from "../components/ui/PageContainer"
import Card from "../components/ui/Card"
import Button from "../components/ui/Button"
import Alert from "../components/ui/Alert"

const userSchema = z.object({
  email: z.string().min(1, "Required").email("Invalid email"),
  name: z
    .string()
    .min(3, "Min 3 characters")
    .max(32, "Max 32 characters")
    .regex(
      /^[a-zA-Z0-9_-]{3,32}$/,
      "Only letters, digits, hyphens, and underscores"
    ),
  admin: z.boolean().optional(),
})

type UserFormData = z.infer<typeof userSchema>

export default function AdminUsers() {
  const { user: currentUser } = useSession()
  const searchSchema = useMemo(() => ({ q: stringParam() }), [])
  const [urlState, setUrlState] = useUrlState(searchSchema)
  const search = urlState.q
  const setSearch = (v: string) => setUrlState({ q: v })
  const [showCreate, setShowCreate] = useState(false)
  const [initialPassword, setInitialPassword] = useState<string | null>(null)
  const [editingUuid, setEditingUuid] = useState<string | null>(null)
  const [resetPassword, setResetPassword] = useState<string | null>(null)
  const [resetConfirm, setResetConfirm] = useState<string | null>(null)
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null)
  const [toast, setToast] = useState<string | null>(null)

  const { data: users, refetch } = $api.useQuery("get", "/admin/users")

  const createForm = useForm<UserFormData>({
    resolver: zodResolver(userSchema),
    defaultValues: { admin: false },
  })
  const editForm = useForm<UserFormData>({
    resolver: zodResolver(userSchema),
  })

  const filtered = (users ?? []).filter(
    (u) =>
      u.name.toLowerCase().includes(search.toLowerCase()) ||
      u.email.toLowerCase().includes(search.toLowerCase())
  )

  async function onCreateUser(data: UserFormData) {
    try {
      const { data: result } = await fetchClient.POST("/admin/users", {
        body: data,
      })
      setInitialPassword(result?.initialPassword ?? null)
      setShowCreate(false)
      createForm.reset()
      await refetch()
    } catch (err) {
      setToast((err as Error).message)
    }
  }

  function startEdit(user: (typeof filtered)[0]) {
    setEditingUuid(user.uuid)
    editForm.reset({
      email: user.email,
      name: user.name,
      admin: user.admin,
    })
  }

  async function onEditUser(data: UserFormData) {
    if (!editingUuid) return
    try {
      await fetchClient.PATCH("/admin/users/{uuid}", {
        params: { path: { uuid: editingUuid } },
        body: data,
      })
      setEditingUuid(null)
      await refetch()
    } catch (err) {
      setToast((err as Error).message)
    }
  }

  async function handleResetPassword(uuid: string) {
    try {
      const resp = await fetch(`/api/admin/users/${uuid}/reset-password`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "X-Requested-With": "cartomancer",
        },
        body: "{}",
      })
      if (!resp.ok) {
        const body = await resp.json().catch(() => null)
        throw new Error(
          (body as { msg?: string } | null)?.msg ?? resp.statusText
        )
      }
      const result: { password: string } = await resp.json()
      setResetPassword(result.password)
    } catch (err) {
      setToast((err as Error).message)
    }
  }

  async function handleConfirmEmail(uuid: string) {
    try {
      const resp = await fetch(`/api/admin/users/${uuid}/confirm-email`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "X-Requested-With": "cartomancer",
        },
        body: "{}",
      })
      if (!resp.ok) {
        const body = await resp.json().catch(() => null)
        throw new Error(
          (body as { msg?: string } | null)?.msg ?? resp.statusText
        )
      }
      await refetch()
    } catch (err) {
      setToast((err as Error).message)
    }
  }

  async function handleDelete(uuid: string) {
    try {
      await fetchClient.DELETE("/admin/users/{uuid}", {
        params: { path: { uuid } },
      })
      setDeleteConfirm(null)
      await refetch()
    } catch (err) {
      setToast((err as Error).message)
    }
  }

  return (
    <PageContainer size="lg">
      <div className="mb-6 flex items-center gap-4">
        <h1 className="text-2xl font-semibold text-text">Admin</h1>
        <Link
          to="/admin/users"
          className="border-b-2 border-primary pb-0.5 text-sm font-medium text-text"
          aria-current="page"
        >
          Users
        </Link>
        <Link
          to="/admin/forecasts"
          className="pb-0.5 text-sm font-medium text-text-muted hover:text-text-secondary transition-colors"
        >
          Forecasts
        </Link>
      </div>

      {(initialPassword || resetPassword) && (
        <Alert variant="warning" className="mb-4">
          <p className="font-medium">
            {initialPassword
              ? "User created. Initial password (shown once):"
              : "Password reset. New password (shown once):"}
          </p>
          <code className="mt-1 block rounded bg-warning-light px-2 py-1 text-sm font-mono">
            {initialPassword ?? resetPassword}
          </code>
          <button
            onClick={() => {
              setInitialPassword(null)
              setResetPassword(null)
            }}
            className="mt-2 cursor-pointer text-sm underline hover:text-text transition-colors"
          >
            Dismiss
          </button>
        </Alert>
      )}

      <div className="mb-4 flex items-center gap-3">
        <input
          type="text"
          placeholder="Search users..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          aria-label="Search users"
          className="w-full max-w-xs rounded border border-border bg-panel px-3 py-2 text-sm text-text placeholder-text-muted focus:border-primary focus:outline-none transition-colors"
        />
        <Button
          variant="primary"
          onClick={() => {
            setShowCreate(!showCreate)
            createForm.reset()
          }}
        >
          {showCreate ? "Cancel" : "Create user"}
        </Button>
      </div>

      {showCreate && (
        <Card className="mb-4 p-4">
          <h3 className="mb-3 text-sm font-medium text-text">Create user</h3>
          <form
            onSubmit={createForm.handleSubmit(onCreateUser)}
            className="flex flex-wrap items-start gap-3"
          >
            <div>
              <input
                type="email"
                placeholder="Email"
                {...createForm.register("email")}
                className="rounded border border-border bg-panel px-3 py-2 text-sm text-text placeholder-text-muted focus:border-primary focus:outline-none transition-colors"
              />
              {createForm.formState.errors.email && (
                <p role="alert" className="mt-1 text-xs text-error">
                  {createForm.formState.errors.email.message}
                </p>
              )}
            </div>
            <div>
              <input
                type="text"
                placeholder="Name"
                {...createForm.register("name")}
                className="rounded border border-border bg-panel px-3 py-2 text-sm text-text placeholder-text-muted focus:border-primary focus:outline-none transition-colors"
              />
              {createForm.formState.errors.name && (
                <p role="alert" className="mt-1 text-xs text-error">
                  {createForm.formState.errors.name.message}
                </p>
              )}
            </div>
            <label className="flex items-center gap-1.5 py-2 text-sm text-text-secondary">
              <input
                type="checkbox"
                {...createForm.register("admin")}
                className="accent-primary"
              />
              Admin
            </label>
            <Button
              type="submit"
              variant="primary"
              disabled={createForm.formState.isSubmitting}
            >
              {createForm.formState.isSubmitting ? "Creating..." : "Create"}
            </Button>
          </form>
        </Card>
      )}

      <Card className="overflow-hidden">
        <table className="w-full text-left text-sm">
          <thead>
            <tr className="border-b border-border text-xs font-medium text-text-muted">
              <th className="px-4 py-3">Name</th>
              <th className="px-4 py-3">Email</th>
              <th className="px-4 py-3">Admin</th>
              <th className="px-4 py-3">Last active</th>
              <th className="px-4 py-3">Actions</th>
            </tr>
          </thead>
          <tbody>
            {filtered.map((u) => (
              <tr key={u.uuid} className="border-b border-border last:border-0">
                {editingUuid === u.uuid ? (
                  <td colSpan={5} className="px-4 py-3">
                    <form
                      onSubmit={editForm.handleSubmit(onEditUser)}
                      className="flex flex-wrap items-start gap-3"
                    >
                      <div>
                        <input
                          type="text"
                          {...editForm.register("name")}
                          className="rounded border border-border bg-panel px-2 py-1 text-sm text-text focus:border-primary focus:outline-none transition-colors"
                        />
                        {editForm.formState.errors.name && (
                          <p role="alert" className="mt-0.5 text-xs text-error">
                            {editForm.formState.errors.name.message}
                          </p>
                        )}
                      </div>
                      <div>
                        <input
                          type="email"
                          {...editForm.register("email")}
                          className="rounded border border-border bg-panel px-2 py-1 text-sm text-text focus:border-primary focus:outline-none transition-colors"
                        />
                        {editForm.formState.errors.email && (
                          <p role="alert" className="mt-0.5 text-xs text-error">
                            {editForm.formState.errors.email.message}
                          </p>
                        )}
                      </div>
                      <label className="flex items-center gap-1.5 py-1 text-sm text-text-secondary">
                        <input
                          type="checkbox"
                          {...editForm.register("admin")}
                          className="accent-primary"
                        />
                        Admin
                      </label>
                      <Button
                        type="submit"
                        variant="primary"
                        disabled={editForm.formState.isSubmitting}
                        className="px-3 py-1"
                      >
                        Save
                      </Button>
                      <Button
                        variant="secondary"
                        onClick={() => setEditingUuid(null)}
                        className="px-3 py-1"
                      >
                        Cancel
                      </Button>
                    </form>
                  </td>
                ) : (
                  <>
                    <td className="px-4 py-3 font-medium text-text">
                      {u.name}
                    </td>
                    <td className="px-4 py-3 text-text-secondary">{u.email}</td>
                    <td className="px-4 py-3 text-text-secondary">
                      {u.admin ? "Yes" : "No"}
                    </td>
                    <td className="px-4 py-3 text-text-muted">
                      {u.lastActiveAt
                        ? u.lastActiveAt.slice(0, 16).replace("T", " ")
                        : "--"}
                    </td>
                    <td className="px-4 py-3">
                      {resetConfirm === u.uuid ? (
                        <span className="flex items-center gap-2">
                          <button
                            onClick={() => {
                              handleResetPassword(u.uuid)
                              setResetConfirm(null)
                            }}
                            className="cursor-pointer text-sm font-medium text-error hover:text-error/80 transition-colors"
                          >
                            Confirm
                          </button>
                          <button
                            onClick={() => setResetConfirm(null)}
                            className="cursor-pointer text-sm text-text-muted hover:text-text-secondary transition-colors"
                          >
                            Cancel
                          </button>
                        </span>
                      ) : deleteConfirm === u.uuid ? (
                        <span className="flex items-center gap-2">
                          <button
                            onClick={() => handleDelete(u.uuid)}
                            className="cursor-pointer text-sm font-medium text-error hover:text-error/80 transition-colors"
                          >
                            Confirm
                          </button>
                          <button
                            onClick={() => setDeleteConfirm(null)}
                            className="cursor-pointer text-sm text-text-muted hover:text-text-secondary transition-colors"
                          >
                            Cancel
                          </button>
                        </span>
                      ) : (
                        <span className="flex items-center gap-3">
                          <button
                            onClick={() => startEdit(u)}
                            className="cursor-pointer text-sm text-text-secondary hover:text-text transition-colors"
                          >
                            Edit
                          </button>
                          {u.uuid !== currentUser?.uuid && (
                            <button
                              onClick={() => setResetConfirm(u.uuid)}
                              className="cursor-pointer text-sm text-text-secondary hover:text-text transition-colors"
                            >
                              Reset pw
                            </button>
                          )}
                          {u.hasPendingEmailVerification && (
                            <button
                              onClick={() => handleConfirmEmail(u.uuid)}
                              className="cursor-pointer text-sm text-text-secondary hover:text-text transition-colors"
                            >
                              Confirm email
                            </button>
                          )}
                          <button
                            onClick={() => setDeleteConfirm(u.uuid)}
                            className="cursor-pointer text-sm text-error hover:text-error/80 transition-colors"
                          >
                            Delete
                          </button>
                        </span>
                      )}
                    </td>
                  </>
                )}
              </tr>
            ))}
            {filtered.length === 0 && (
              <tr>
                <td
                  colSpan={5}
                  className="px-4 py-6 text-center text-sm text-text-muted"
                >
                  No users found.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </Card>

      {toast && <Toast message={toast} onDismiss={() => setToast(null)} />}
    </PageContainer>
  )
}
