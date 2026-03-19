import { useState } from "react"
import { Link } from "react-router-dom"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import { $api, fetchClient } from "../api/client"
import { useSession } from "../context/SessionContext"
import Toast from "../components/Toast"

const userSchema = z.object({
  email: z.string().min(1, "Required").email("Invalid email"),
  name: z
    .string()
    .min(3, "Min 3 characters")
    .max(32, "Max 32 characters")
    .regex(/^[a-zA-Z_-]{3,32}$/, "Only letters, hyphens, and underscores"),
  admin: z.boolean().optional(),
})

type UserFormData = z.infer<typeof userSchema>

export default function AdminUsers() {
  const { user: currentUser } = useSession()
  const [search, setSearch] = useState("")
  const [showCreate, setShowCreate] = useState(false)
  const [initialPassword, setInitialPassword] = useState<string | null>(null)
  const [editingUuid, setEditingUuid] = useState<string | null>(null)
  const [resetPassword, setResetPassword] = useState<string | null>(null)
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
          "X-Requested-With": "detour",
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
          "X-Requested-With": "detour",
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
    <div className="mx-auto max-w-4xl px-4 py-8">
      <div className="mb-6 flex items-center gap-4">
        <h1 className="text-2xl font-semibold text-gray-900">Admin</h1>
        <Link
          to="/admin/users"
          className="border-b-2 border-gray-900 pb-0.5 text-sm font-medium text-gray-900"
        >
          Users
        </Link>
        <Link
          to="/admin/forecasts"
          className="pb-0.5 text-sm font-medium text-gray-500 hover:text-gray-700"
        >
          Forecasts
        </Link>
      </div>

      {(initialPassword || resetPassword) && (
        <div className="mb-4 rounded-lg border border-yellow-300 bg-yellow-50 p-4">
          <p className="text-sm font-medium text-yellow-800">
            {initialPassword
              ? "User created. Initial password (shown once):"
              : "Password reset. New password (shown once):"}
          </p>
          <code className="mt-1 block rounded bg-yellow-100 px-2 py-1 text-sm font-mono text-yellow-900">
            {initialPassword ?? resetPassword}
          </code>
          <button
            onClick={() => {
              setInitialPassword(null)
              setResetPassword(null)
            }}
            className="mt-2 cursor-pointer text-sm text-yellow-700 underline hover:text-yellow-900"
          >
            Dismiss
          </button>
        </div>
      )}

      <div className="mb-4 flex items-center gap-3">
        <input
          type="text"
          placeholder="Search users..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="w-full max-w-xs rounded border border-gray-300 px-3 py-2 text-sm focus:border-gray-500 focus:outline-none"
        />
        <button
          onClick={() => {
            setShowCreate(!showCreate)
            createForm.reset()
          }}
          className="cursor-pointer rounded bg-gray-900 px-4 py-2 text-sm font-medium text-white hover:bg-gray-700"
        >
          {showCreate ? "Cancel" : "Create user"}
        </button>
      </div>

      {showCreate && (
        <div className="mb-4 rounded-lg border border-gray-200 bg-white p-4">
          <h3 className="mb-3 text-sm font-medium text-gray-900">
            Create user
          </h3>
          <form
            onSubmit={createForm.handleSubmit(onCreateUser)}
            className="flex flex-wrap items-start gap-3"
          >
            <div>
              <input
                type="email"
                placeholder="Email"
                {...createForm.register("email")}
                className="rounded border border-gray-300 px-3 py-2 text-sm focus:border-gray-500 focus:outline-none"
              />
              {createForm.formState.errors.email && (
                <p className="mt-1 text-xs text-red-600">
                  {createForm.formState.errors.email.message}
                </p>
              )}
            </div>
            <div>
              <input
                type="text"
                placeholder="Name"
                {...createForm.register("name")}
                className="rounded border border-gray-300 px-3 py-2 text-sm focus:border-gray-500 focus:outline-none"
              />
              {createForm.formState.errors.name && (
                <p className="mt-1 text-xs text-red-600">
                  {createForm.formState.errors.name.message}
                </p>
              )}
            </div>
            <label className="flex items-center gap-1.5 py-2 text-sm text-gray-700">
              <input type="checkbox" {...createForm.register("admin")} />
              Admin
            </label>
            <button
              type="submit"
              disabled={createForm.formState.isSubmitting}
              className="cursor-pointer rounded bg-gray-900 px-4 py-2 text-sm font-medium text-white hover:bg-gray-700 disabled:cursor-not-allowed disabled:opacity-50"
            >
              {createForm.formState.isSubmitting ? "Creating..." : "Create"}
            </button>
          </form>
        </div>
      )}

      <div className="rounded-lg border border-gray-200 bg-white">
        <table className="w-full text-left text-sm">
          <thead>
            <tr className="border-b border-gray-200 text-xs font-medium text-gray-500">
              <th className="px-4 py-3">Name</th>
              <th className="px-4 py-3">Email</th>
              <th className="px-4 py-3">Admin</th>
              <th className="px-4 py-3">Last active</th>
              <th className="px-4 py-3">Actions</th>
            </tr>
          </thead>
          <tbody>
            {filtered.map((u) => (
              <tr
                key={u.uuid}
                className="border-b border-gray-100 last:border-0"
              >
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
                          className="rounded border border-gray-300 px-2 py-1 text-sm focus:border-gray-500 focus:outline-none"
                        />
                        {editForm.formState.errors.name && (
                          <p className="mt-0.5 text-xs text-red-600">
                            {editForm.formState.errors.name.message}
                          </p>
                        )}
                      </div>
                      <div>
                        <input
                          type="email"
                          {...editForm.register("email")}
                          className="rounded border border-gray-300 px-2 py-1 text-sm focus:border-gray-500 focus:outline-none"
                        />
                        {editForm.formState.errors.email && (
                          <p className="mt-0.5 text-xs text-red-600">
                            {editForm.formState.errors.email.message}
                          </p>
                        )}
                      </div>
                      <label className="flex items-center gap-1.5 py-1 text-sm text-gray-700">
                        <input
                          type="checkbox"
                          {...editForm.register("admin")}
                        />
                        Admin
                      </label>
                      <button
                        type="submit"
                        disabled={editForm.formState.isSubmitting}
                        className="cursor-pointer rounded bg-gray-900 px-3 py-1 text-sm font-medium text-white hover:bg-gray-700 disabled:opacity-50"
                      >
                        Save
                      </button>
                      <button
                        type="button"
                        onClick={() => setEditingUuid(null)}
                        className="cursor-pointer rounded border border-gray-300 px-3 py-1 text-sm text-gray-700 hover:bg-gray-50"
                      >
                        Cancel
                      </button>
                    </form>
                  </td>
                ) : (
                  <>
                    <td className="px-4 py-3 font-medium text-gray-900">
                      {u.name}
                    </td>
                    <td className="px-4 py-3 text-gray-600">{u.email}</td>
                    <td className="px-4 py-3 text-gray-600">
                      {u.admin ? "Yes" : "No"}
                    </td>
                    <td className="px-4 py-3 text-gray-500">
                      {u.lastActiveAt
                        ? u.lastActiveAt.slice(0, 16).replace("T", " ")
                        : "--"}
                    </td>
                    <td className="px-4 py-3">
                      {deleteConfirm === u.uuid ? (
                        <span className="flex items-center gap-2">
                          <button
                            onClick={() => handleDelete(u.uuid)}
                            className="cursor-pointer text-sm font-medium text-red-600 hover:text-red-800"
                          >
                            Confirm
                          </button>
                          <button
                            onClick={() => setDeleteConfirm(null)}
                            className="cursor-pointer text-sm text-gray-500 hover:text-gray-700"
                          >
                            Cancel
                          </button>
                        </span>
                      ) : (
                        <span className="flex items-center gap-3">
                          <button
                            onClick={() => startEdit(u)}
                            className="cursor-pointer text-sm text-gray-600 hover:text-gray-900"
                          >
                            Edit
                          </button>
                          {u.uuid !== currentUser?.uuid && (
                            <button
                              onClick={() => handleResetPassword(u.uuid)}
                              className="cursor-pointer text-sm text-gray-600 hover:text-gray-900"
                            >
                              Reset pw
                            </button>
                          )}
                          <button
                            onClick={() => handleConfirmEmail(u.uuid)}
                            className="cursor-pointer text-sm text-gray-600 hover:text-gray-900"
                          >
                            Confirm email
                          </button>
                          <button
                            onClick={() => setDeleteConfirm(u.uuid)}
                            className="cursor-pointer text-sm text-red-600 hover:text-red-800"
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
                  className="px-4 py-6 text-center text-sm text-gray-500"
                >
                  No users found.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      {toast && <Toast message={toast} onDismiss={() => setToast(null)} />}
    </div>
  )
}
