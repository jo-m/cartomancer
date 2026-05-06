import { useMemo, useState } from "react"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import { $api, fetchClient } from "../api/client"
import { useSession } from "../context/SessionContext"
import useDocumentTitle from "../hooks/useDocumentTitle"
import { useUrlState, stringParam } from "../hooks/useUrlState"
import Toast from "../components/Toast"
import useToast from "../hooks/useToast"
import PageContainer from "../components/ui/PageContainer"
import Card from "../components/ui/Card"
import Button from "../components/ui/Button"
import Alert from "../components/ui/Alert"
import Input from "../components/ui/Input"
import CopyIdCell from "../components/CopyIdCell"
import AdminTabs from "../components/admin/AdminTabs"
import AdminCard, {
  AdminCardField,
  AdminCardFooter,
  AdminCardHeader,
} from "../components/admin/AdminCard"
import TimeAgo from "../components/TimeAgo"

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
  useDocumentTitle("Users")
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
  const { toast, showToast, dismissToast } = useToast()

  const {
    data: users,
    isLoading,
    refetch,
  } = $api.useQuery("get", "/admin/users")

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
      showToast((err as Error).message)
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
      showToast("User updated.", "success")
    } catch (err) {
      showToast((err as Error).message)
    }
  }

  async function handleResetPassword(uuid: string) {
    try {
      const { data } = await fetchClient.POST(
        "/admin/users/{uuid}/reset-password",
        { params: { path: { uuid } }, body: {} }
      )
      setResetPassword(data?.password ?? null)
    } catch (err) {
      showToast((err as Error).message)
    }
  }

  async function handleConfirmEmail(uuid: string) {
    try {
      await fetchClient.POST("/admin/users/{uuid}/confirm-email", {
        params: { path: { uuid } },
        body: {},
      })
      await refetch()
    } catch (err) {
      showToast((err as Error).message)
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
      showToast((err as Error).message)
    }
  }

  function renderEditForm() {
    return (
      <form
        onSubmit={editForm.handleSubmit(onEditUser)}
        className="flex flex-col items-stretch gap-3 sm:flex-row sm:flex-wrap sm:items-start"
      >
        <Input
          type="text"
          {...editForm.register("name")}
          error={editForm.formState.errors.name?.message}
        />
        <Input
          type="email"
          {...editForm.register("email")}
          error={editForm.formState.errors.email?.message}
        />
        <label className="flex items-center gap-1.5 py-1 text-sm text-text-secondary">
          <input
            type="checkbox"
            {...editForm.register("admin")}
            className="accent-primary"
          />
          Admin
        </label>
        <div className="flex flex-wrap gap-2">
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
        </div>
      </form>
    )
  }

  function renderActions(u: (typeof filtered)[0]) {
    if (resetConfirm === u.uuid) {
      return (
        <>
          <button
            onClick={() => {
              handleResetPassword(u.uuid)
              setResetConfirm(null)
            }}
            className="cursor-pointer text-sm font-medium text-error transition-colors hover:text-error/80"
          >
            Confirm
          </button>
          <button
            onClick={() => setResetConfirm(null)}
            className="cursor-pointer text-sm text-text-muted transition-colors hover:text-text-secondary"
          >
            Cancel
          </button>
        </>
      )
    }
    if (deleteConfirm === u.uuid) {
      return (
        <>
          <button
            onClick={() => handleDelete(u.uuid)}
            className="cursor-pointer text-sm font-medium text-error transition-colors hover:text-error/80"
          >
            Confirm
          </button>
          <button
            onClick={() => setDeleteConfirm(null)}
            className="cursor-pointer text-sm text-text-muted transition-colors hover:text-text-secondary"
          >
            Cancel
          </button>
        </>
      )
    }
    return (
      <>
        <button
          onClick={() => startEdit(u)}
          className="cursor-pointer text-sm text-text-secondary transition-colors hover:text-text"
        >
          Edit
        </button>
        {u.uuid !== currentUser?.uuid && (
          <button
            onClick={() => setResetConfirm(u.uuid)}
            className="cursor-pointer text-sm text-text-secondary transition-colors hover:text-text"
          >
            Reset pw
          </button>
        )}
        {u.hasPendingEmailVerification && (
          <button
            onClick={() => handleConfirmEmail(u.uuid)}
            className="cursor-pointer text-sm text-text-secondary transition-colors hover:text-text"
          >
            Confirm email
          </button>
        )}
        <button
          onClick={() => setDeleteConfirm(u.uuid)}
          className="cursor-pointer text-sm text-error transition-colors hover:text-error/80"
        >
          Delete
        </button>
      </>
    )
  }

  return (
    <PageContainer size="2xl">
      <AdminTabs current="users" />

      {(initialPassword || resetPassword) && (
        <Alert variant="warning" className="mb-4">
          <p className="font-medium">
            {initialPassword
              ? "User created. Initial password (shown once):"
              : "Password reset. New password (shown once):"}
          </p>
          <code className="mt-1 block rounded bg-warning-light px-2 py-1 font-mono text-sm">
            {initialPassword ?? resetPassword}
          </code>
          <button
            onClick={() => {
              setInitialPassword(null)
              setResetPassword(null)
            }}
            className="mt-2 cursor-pointer text-sm underline transition-colors hover:text-text"
          >
            Dismiss
          </button>
        </Alert>
      )}

      <div className="mb-4 flex flex-wrap items-center gap-3">
        <Input
          type="text"
          placeholder="Search users..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          aria-label="Search users"
          className="max-w-xs"
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
            className="flex flex-col items-stretch gap-3 sm:flex-row sm:flex-wrap sm:items-start"
          >
            <Input
              type="email"
              placeholder="Email"
              {...createForm.register("email")}
              error={createForm.formState.errors.email?.message}
            />
            <Input
              type="text"
              placeholder="Name"
              {...createForm.register("name")}
              error={createForm.formState.errors.name?.message}
            />
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
              className="self-start"
            >
              {createForm.formState.isSubmitting ? "Creating..." : "Create"}
            </Button>
          </form>
        </Card>
      )}

      <Card className="hidden overflow-x-auto md:block">
        <table className="w-full text-left text-sm">
          <thead>
            <tr className="border-b border-border text-xs font-medium text-text-muted">
              <th className="px-4 py-3">ID</th>
              <th className="px-4 py-3">Name</th>
              <th className="px-4 py-3">Email</th>
              <th className="px-4 py-3">Admin</th>
              <th className="px-4 py-3">Tracks</th>
              <th className="px-4 py-3">Created</th>
              <th className="px-4 py-3">Last active</th>
              <th className="px-4 py-3">Actions</th>
            </tr>
          </thead>
          <tbody>
            {filtered.map((u) => (
              <tr key={u.uuid} className="border-b border-border last:border-0">
                {editingUuid === u.uuid ? (
                  <td colSpan={8} className="px-4 py-3">
                    {renderEditForm()}
                  </td>
                ) : (
                  <>
                    <td className="px-4 py-3">
                      <CopyIdCell
                        id={u.uuid}
                        onCopied={() =>
                          showToast("Copied to clipboard", "success")
                        }
                      />
                    </td>
                    <td className="px-4 py-3 font-medium text-text">
                      {u.name}
                    </td>
                    <td className="px-4 py-3 text-text-secondary">{u.email}</td>
                    <td className="px-4 py-3 text-text-secondary">
                      {u.admin ? "Yes" : "No"}
                    </td>
                    <td className="px-4 py-3 text-text-secondary">
                      {u.trackCount}
                    </td>
                    <td className="px-4 py-3 text-text-muted">
                      <TimeAgo iso={u.createdAt} />
                    </td>
                    <td className="px-4 py-3 text-text-muted">
                      {u.lastActiveAt ? <TimeAgo iso={u.lastActiveAt} /> : "--"}
                    </td>
                    <td className="px-4 py-3">
                      <span className="flex items-center gap-3">
                        {renderActions(u)}
                      </span>
                    </td>
                  </>
                )}
              </tr>
            ))}
            {filtered.length === 0 && (
              <tr>
                <td
                  colSpan={8}
                  className="px-4 py-6 text-center text-sm text-text-muted"
                >
                  {isLoading ? "Loading..." : "No users found."}
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </Card>

      <div className="space-y-3 md:hidden">
        {filtered.map((u) => (
          <AdminCard key={u.uuid}>
            {editingUuid === u.uuid ? (
              renderEditForm()
            ) : (
              <>
                <AdminCardHeader>
                  <span className="font-medium text-text">{u.name}</span>
                  <CopyIdCell
                    id={u.uuid}
                    onCopied={() => showToast("Copied to clipboard", "success")}
                  />
                </AdminCardHeader>
                <AdminCardField label="Email">{u.email}</AdminCardField>
                <AdminCardField label="Admin">
                  {u.admin ? "Yes" : "No"}
                </AdminCardField>
                <AdminCardField label="Tracks">{u.trackCount}</AdminCardField>
                <AdminCardField label="Created">
                  <TimeAgo iso={u.createdAt} />
                </AdminCardField>
                <AdminCardField label="Last active">
                  {u.lastActiveAt ? <TimeAgo iso={u.lastActiveAt} /> : "--"}
                </AdminCardField>
                <AdminCardFooter>{renderActions(u)}</AdminCardFooter>
              </>
            )}
          </AdminCard>
        ))}
        {filtered.length === 0 && (
          <Card className="px-4 py-6 text-center text-sm text-text-muted">
            {isLoading ? "Loading..." : "No users found."}
          </Card>
        )}
      </div>

      {toast && (
        <Toast
          key={toast.key}
          message={toast.message}
          variant={toast.variant}
          onDismiss={dismissToast}
        />
      )}
    </PageContainer>
  )
}
