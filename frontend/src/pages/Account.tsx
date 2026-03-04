import { useEffect, useState } from "react"
import { useNavigate } from "react-router-dom"
import { useSession } from "../context/SessionContext"
import { updateMe, changePassword, deleteMe } from "../api/client"

export default function Account() {
  const { user, loading, setUser, logout } = useSession()
  const navigate = useNavigate()

  const [name, setName] = useState(user?.name ?? "")
  const [profileError, setProfileError] = useState<string | null>(null)
  const [profileSuccess, setProfileSuccess] = useState(false)
  const [profileLoading, setProfileLoading] = useState(false)

  const [oldPassword, setOldPassword] = useState("")
  const [newPassword, setNewPassword] = useState("")
  const [passwordError, setPasswordError] = useState<string | null>(null)
  const [passwordSuccess, setPasswordSuccess] = useState(false)
  const [passwordLoading, setPasswordLoading] = useState(false)

  const [confirmDelete, setConfirmDelete] = useState(false)
  const [deleteError, setDeleteError] = useState<string | null>(null)
  const [deleteLoading, setDeleteLoading] = useState(false)

  useEffect(() => {
    if (!loading && !user) navigate("/login")
  }, [user, loading, navigate])

  useEffect(() => {
    if (user) setName(user.name)
  }, [user])

  if (loading || !user) return null

  async function handleUpdateProfile(e: React.FormEvent) {
    e.preventDefault()
    setProfileError(null)
    setProfileSuccess(false)
    setProfileLoading(true)
    try {
      const updated = await updateMe({ name })
      setUser(updated)
      setProfileSuccess(true)
    } catch (err) {
      setProfileError(err instanceof Error ? err.message : "Update failed")
    } finally {
      setProfileLoading(false)
    }
  }

  async function handleChangePassword(e: React.FormEvent) {
    e.preventDefault()
    setPasswordError(null)
    setPasswordSuccess(false)
    setPasswordLoading(true)
    try {
      await changePassword({ oldPassword, newPassword })
      setPasswordSuccess(true)
      setOldPassword("")
      setNewPassword("")
    } catch (err) {
      setPasswordError(
        err instanceof Error ? err.message : "Failed to change password"
      )
    } finally {
      setPasswordLoading(false)
    }
  }

  async function handleDeleteAccount() {
    setDeleteError(null)
    setDeleteLoading(true)
    try {
      await deleteMe()
      await logout()
      navigate("/login")
    } catch (err) {
      setDeleteError(
        err instanceof Error ? err.message : "Failed to delete account"
      )
      setDeleteLoading(false)
    }
  }

  return (
    <div className="mx-auto max-w-lg px-4 py-8">
      <h1 className="mb-8 text-2xl font-semibold text-gray-900">Account</h1>

      <section className="mb-6 rounded-lg border border-gray-200 bg-white p-6">
        <h2 className="mb-4 text-base font-medium text-gray-900">Profile</h2>
        <form onSubmit={handleUpdateProfile} className="space-y-4">
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
              value={name}
              onChange={(e) => setName(e.target.value)}
              required
              className="w-full rounded border border-gray-300 px-3 py-2 text-sm focus:border-gray-500 focus:outline-none"
            />
          </div>
          {profileError && (
            <p className="text-sm text-red-600">{profileError}</p>
          )}
          {profileSuccess && (
            <p className="text-sm text-green-600">Profile updated.</p>
          )}
          <button
            type="submit"
            disabled={profileLoading}
            className="cursor-pointer rounded bg-gray-900 px-4 py-2 text-sm font-medium text-white hover:bg-gray-700 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {profileLoading ? "Saving…" : "Save"}
          </button>
        </form>
      </section>

      <section className="mb-6 rounded-lg border border-gray-200 bg-white p-6">
        <h2 className="mb-4 text-base font-medium text-gray-900">
          Change Password
        </h2>
        <form onSubmit={handleChangePassword} className="space-y-4">
          <div>
            <label className="mb-1 block text-sm font-medium text-gray-700">
              Current password
            </label>
            <input
              type="password"
              value={oldPassword}
              onChange={(e) => setOldPassword(e.target.value)}
              required
              autoComplete="current-password"
              className="w-full rounded border border-gray-300 px-3 py-2 text-sm focus:border-gray-500 focus:outline-none"
            />
          </div>
          <div>
            <label className="mb-1 block text-sm font-medium text-gray-700">
              New password
            </label>
            <input
              type="password"
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
              required
              autoComplete="new-password"
              className="w-full rounded border border-gray-300 px-3 py-2 text-sm focus:border-gray-500 focus:outline-none"
            />
          </div>
          {passwordError && (
            <p className="text-sm text-red-600">{passwordError}</p>
          )}
          {passwordSuccess && (
            <p className="text-sm text-green-600">Password changed.</p>
          )}
          <button
            type="submit"
            disabled={passwordLoading}
            className="cursor-pointer rounded bg-gray-900 px-4 py-2 text-sm font-medium text-white hover:bg-gray-700 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {passwordLoading ? "Saving…" : "Change password"}
          </button>
        </form>
      </section>

      <section className="rounded-lg border border-red-200 bg-white p-6">
        <h2 className="mb-4 text-base font-medium text-red-800">Danger Zone</h2>
        {deleteError && (
          <p className="mb-3 text-sm text-red-600">{deleteError}</p>
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
                disabled={deleteLoading}
                className="cursor-pointer rounded bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700 disabled:cursor-not-allowed disabled:opacity-50"
              >
                {deleteLoading ? "Deleting…" : "Yes, delete my account"}
              </button>
              <button
                onClick={() => setConfirmDelete(false)}
                disabled={deleteLoading}
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
