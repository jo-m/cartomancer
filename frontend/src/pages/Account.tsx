import { useEffect, useState } from "react"
import { useNavigate } from "react-router-dom"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import { useSession } from "../context/SessionContext"
import { $api } from "../api/client"

const profileSchema = z.object({
  name: z.string().min(1, "Required"),
})

const passwordSchema = z.object({
  oldPassword: z.string().min(1, "Required"),
  newPassword: z.string().min(1, "Required"),
})

type ProfileData = z.infer<typeof profileSchema>
type PasswordData = z.infer<typeof passwordSchema>

export default function Account() {
  const { user, loading, setUser, logout } = useSession()
  const navigate = useNavigate()
  const [confirmDelete, setConfirmDelete] = useState(false)

  const updateMeMutation = $api.useMutation("patch", "/account")
  const changePasswordMutation = $api.useMutation(
    "post",
    "/account/change-password"
  )
  const deleteMeMutation = $api.useMutation("delete", "/account")

  const profileForm = useForm<ProfileData>({
    resolver: zodResolver(profileSchema),
    values: { name: user?.name ?? "" },
  })
  const passwordForm = useForm<PasswordData>({
    resolver: zodResolver(passwordSchema),
  })

  useEffect(() => {
    if (!loading && !user) navigate("/login")
  }, [user, loading, navigate])

  if (loading || !user) return null

  async function onUpdateProfile(data: ProfileData) {
    try {
      const updated = await updateMeMutation.mutateAsync({ body: data })
      setUser(updated)
    } catch {
      // error displayed via updateMeMutation.error
    }
  }

  async function onChangePassword(data: PasswordData) {
    try {
      await changePasswordMutation.mutateAsync({ body: data })
      passwordForm.reset()
    } catch {
      // error displayed via changePasswordMutation.error
    }
  }

  async function handleDeleteAccount() {
    try {
      await deleteMeMutation.mutateAsync({})
      await logout()
      navigate("/login")
    } catch {
      // error displayed via deleteMeMutation.error
    }
  }

  return (
    <div className="mx-auto max-w-lg px-4 py-8">
      <h1 className="mb-8 text-2xl font-semibold text-gray-900">Account</h1>

      <section className="mb-6 rounded-lg border border-gray-200 bg-white p-6">
        <h2 className="mb-4 text-base font-medium text-gray-900">Profile</h2>
        <form
          onSubmit={profileForm.handleSubmit(onUpdateProfile)}
          className="space-y-4"
        >
          <div>
            <label className="mb-1 block text-sm font-medium text-gray-700">
              Email
            </label>
            <p className="text-sm text-gray-500">{user.email}</p>
          </div>
          <div>
            <label className="mb-1 block text-sm font-medium text-gray-700">
              Name
            </label>
            <input
              type="text"
              {...profileForm.register("name")}
              className="w-full rounded border border-gray-300 px-3 py-2 text-sm focus:border-gray-500 focus:outline-none"
            />
            {profileForm.formState.errors.name && (
              <p className="mt-1 text-sm text-red-600">
                {profileForm.formState.errors.name.message}
              </p>
            )}
          </div>
          {updateMeMutation.error && (
            <p className="text-sm text-red-600">
              {(updateMeMutation.error as unknown as Error).message}
            </p>
          )}
          {updateMeMutation.isSuccess && (
            <p className="text-sm text-green-600">Profile updated.</p>
          )}
          <button
            type="submit"
            disabled={profileForm.formState.isSubmitting}
            className="cursor-pointer rounded bg-gray-900 px-4 py-2 text-sm font-medium text-white hover:bg-gray-700 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {profileForm.formState.isSubmitting ? "Saving…" : "Save"}
          </button>
        </form>
      </section>

      <section className="mb-6 rounded-lg border border-gray-200 bg-white p-6">
        <h2 className="mb-4 text-base font-medium text-gray-900">
          Change Password
        </h2>
        <form
          onSubmit={passwordForm.handleSubmit(onChangePassword)}
          className="space-y-4"
        >
          <div>
            <label className="mb-1 block text-sm font-medium text-gray-700">
              Current password
            </label>
            <input
              type="password"
              autoComplete="current-password"
              {...passwordForm.register("oldPassword")}
              className="w-full rounded border border-gray-300 px-3 py-2 text-sm focus:border-gray-500 focus:outline-none"
            />
            {passwordForm.formState.errors.oldPassword && (
              <p className="mt-1 text-sm text-red-600">
                {passwordForm.formState.errors.oldPassword.message}
              </p>
            )}
          </div>
          <div>
            <label className="mb-1 block text-sm font-medium text-gray-700">
              New password
            </label>
            <input
              type="password"
              autoComplete="new-password"
              {...passwordForm.register("newPassword")}
              className="w-full rounded border border-gray-300 px-3 py-2 text-sm focus:border-gray-500 focus:outline-none"
            />
            {passwordForm.formState.errors.newPassword && (
              <p className="mt-1 text-sm text-red-600">
                {passwordForm.formState.errors.newPassword.message}
              </p>
            )}
          </div>
          {changePasswordMutation.error && (
            <p className="text-sm text-red-600">
              {(changePasswordMutation.error as unknown as Error).message}
            </p>
          )}
          {changePasswordMutation.isSuccess && (
            <p className="text-sm text-green-600">Password changed.</p>
          )}
          <button
            type="submit"
            disabled={passwordForm.formState.isSubmitting}
            className="cursor-pointer rounded bg-gray-900 px-4 py-2 text-sm font-medium text-white hover:bg-gray-700 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {passwordForm.formState.isSubmitting
              ? "Saving…"
              : "Change password"}
          </button>
        </form>
      </section>

      <section className="rounded-lg border border-red-200 bg-white p-6">
        <h2 className="mb-4 text-base font-medium text-red-800">Danger Zone</h2>
        {deleteMeMutation.error && (
          <p className="mb-3 text-sm text-red-600">
            {(deleteMeMutation.error as unknown as Error).message}
          </p>
        )}
        {confirmDelete ? (
          <div>
            <p className="mb-3 text-sm text-gray-700">
              This will permanently delete your account and all your data. This
              cannot be undone.
            </p>
            <div className="flex gap-2">
              <button
                onClick={handleDeleteAccount}
                disabled={deleteMeMutation.isPending}
                className="cursor-pointer rounded bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700 disabled:cursor-not-allowed disabled:opacity-50"
              >
                {deleteMeMutation.isPending
                  ? "Deleting…"
                  : "Yes, delete my account"}
              </button>
              <button
                onClick={() => setConfirmDelete(false)}
                disabled={deleteMeMutation.isPending}
                className="cursor-pointer rounded border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-50"
              >
                Cancel
              </button>
            </div>
          </div>
        ) : (
          <button
            onClick={() => setConfirmDelete(true)}
            className="cursor-pointer rounded border border-red-300 px-4 py-2 text-sm font-medium text-red-700 hover:bg-red-50"
          >
            Delete account
          </button>
        )}
      </section>
    </div>
  )
}
