import { useState } from "react"
import { useNavigate } from "react-router-dom"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import { useSession } from "../context/SessionContext"
import { $api } from "../api/client"
import Toast from "../components/Toast"
import useToast from "../hooks/useToast"
import PageContainer from "../components/ui/PageContainer"
import Card from "../components/ui/Card"
import Input from "../components/ui/Input"
import Button from "../components/ui/Button"

const profileSchema = z.object({
  name: z.string().min(1, "Required"),
})

const changeEmailSchema = z.object({
  newEmail: z.string().min(1, "Required").email("Invalid email"),
  password: z.string().min(1, "Required"),
})

const passwordSchema = z.object({
  oldPassword: z.string().min(1, "Required"),
  newPassword: z.string().min(1, "Required"),
})

type ProfileData = z.infer<typeof profileSchema>
type ChangeEmailData = z.infer<typeof changeEmailSchema>
type PasswordData = z.infer<typeof passwordSchema>

export default function Account() {
  const { user, invalidateSession, logout } = useSession()
  const navigate = useNavigate()
  const [confirmDelete, setConfirmDelete] = useState(false)
  const { toast, showToast, dismissToast } = useToast()

  const updateMeMutation = $api.useMutation("patch", "/account")
  const rotateAvatarMutation = $api.useMutation(
    "post",
    "/account/rotate-avatar"
  )
  const changeEmailMutation = $api.useMutation("post", "/account/change-email")
  const changePasswordMutation = $api.useMutation(
    "post",
    "/account/change-password"
  )
  const deleteMeMutation = $api.useMutation("delete", "/account")

  const profileForm = useForm<ProfileData>({
    resolver: zodResolver(profileSchema),
    values: { name: user?.name ?? "" },
  })
  const changeEmailForm = useForm<ChangeEmailData>({
    resolver: zodResolver(changeEmailSchema),
  })
  const passwordForm = useForm<PasswordData>({
    resolver: zodResolver(passwordSchema),
  })

  async function handleRotateAvatar() {
    try {
      await rotateAvatarMutation.mutateAsync({})
      await invalidateSession()
    } catch (err) {
      showToast((err as Error).message)
    }
  }

  async function onUpdateProfile(data: ProfileData) {
    try {
      await updateMeMutation.mutateAsync({ body: data })
      await invalidateSession()
      showToast("Profile updated.", "success")
    } catch (err) {
      showToast((err as Error).message)
    }
  }

  async function onChangeEmail(data: ChangeEmailData) {
    try {
      await changeEmailMutation.mutateAsync({ body: data })
      changeEmailForm.reset()
      showToast("Confirmation email sent. Check your inbox.", "success")
    } catch (err) {
      showToast((err as Error).message)
    }
  }

  async function onChangePassword(data: PasswordData) {
    try {
      await changePasswordMutation.mutateAsync({ body: data })
      passwordForm.reset()
      showToast("Password changed.", "success")
    } catch (err) {
      showToast((err as Error).message)
    }
  }

  async function handleDeleteAccount() {
    try {
      await deleteMeMutation.mutateAsync({})
      await logout()
      navigate("/login")
    } catch (err) {
      showToast((err as Error).message)
    }
  }

  return (
    <PageContainer size="sm">
      <h1 className="mb-8 text-2xl font-semibold text-text">Account</h1>

      <Card className="mb-6 p-6">
        <h2 className="mb-4 text-base font-medium text-text">Avatar</h2>
        <div className="flex items-center gap-4">
          {user && (
            <img
              src={`/api/users/${user.uuid}/avatar?v=${user.avatarSeed}`}
              alt={user.name}
              className="h-16 w-16 rounded-full border border-border"
            />
          )}
          <div>
            <Button
              variant="secondary"
              onClick={handleRotateAvatar}
              disabled={rotateAvatarMutation.isPending}
            >
              {rotateAvatarMutation.isPending ? "Rotating..." : "Rotate avatar"}
            </Button>
          </div>
        </div>
      </Card>

      <Card className="mb-6 p-6">
        <h2 className="mb-4 text-base font-medium text-text">Profile</h2>
        <form
          onSubmit={profileForm.handleSubmit(onUpdateProfile)}
          className="space-y-4"
        >
          <div>
            <label className="mb-1 block text-sm font-medium text-text-secondary">
              Email
            </label>
            <p className="text-sm text-text-muted">{user?.email}</p>
          </div>
          <Input
            label="Name"
            type="text"
            error={profileForm.formState.errors.name?.message}
            {...profileForm.register("name")}
          />
          <Button
            type="submit"
            variant="primary"
            disabled={profileForm.formState.isSubmitting}
          >
            {profileForm.formState.isSubmitting ? "Saving..." : "Save"}
          </Button>
        </form>
      </Card>

      <Card className="mb-6 p-6">
        <h2 className="mb-4 text-base font-medium text-text">Change Email</h2>
        <form
          onSubmit={changeEmailForm.handleSubmit(onChangeEmail)}
          className="space-y-4"
        >
          <Input
            label="New email"
            type="email"
            autoComplete="email"
            error={changeEmailForm.formState.errors.newEmail?.message}
            {...changeEmailForm.register("newEmail")}
          />
          <Input
            label="Current password"
            type="password"
            autoComplete="current-password"
            error={changeEmailForm.formState.errors.password?.message}
            {...changeEmailForm.register("password")}
          />
          <Button
            type="submit"
            variant="primary"
            disabled={changeEmailForm.formState.isSubmitting}
          >
            {changeEmailForm.formState.isSubmitting
              ? "Sending..."
              : "Change email"}
          </Button>
        </form>
      </Card>

      <Card className="mb-6 p-6">
        <h2 className="mb-4 text-base font-medium text-text">
          Change Password
        </h2>
        <form
          onSubmit={passwordForm.handleSubmit(onChangePassword)}
          className="space-y-4"
        >
          <Input
            label="Current password"
            type="password"
            autoComplete="current-password"
            error={passwordForm.formState.errors.oldPassword?.message}
            {...passwordForm.register("oldPassword")}
          />
          <Input
            label="New password"
            type="password"
            autoComplete="new-password"
            error={passwordForm.formState.errors.newPassword?.message}
            {...passwordForm.register("newPassword")}
          />
          <Button
            type="submit"
            variant="primary"
            disabled={passwordForm.formState.isSubmitting}
          >
            {passwordForm.formState.isSubmitting
              ? "Saving..."
              : "Change password"}
          </Button>
        </form>
      </Card>

      <Card className="border-error-border p-6">
        <h2 className="mb-4 text-base font-medium text-error">Danger Zone</h2>
        {confirmDelete ? (
          <div>
            <p className="mb-3 text-sm text-text-secondary">
              This will permanently delete your account and all your data. This
              cannot be undone.
            </p>
            <div className="flex gap-2">
              <Button
                variant="danger"
                onClick={handleDeleteAccount}
                disabled={deleteMeMutation.isPending}
                className="bg-error text-primary-text hover:bg-error/90"
              >
                {deleteMeMutation.isPending
                  ? "Deleting..."
                  : "Yes, delete my account"}
              </Button>
              <Button
                variant="secondary"
                onClick={() => setConfirmDelete(false)}
                disabled={deleteMeMutation.isPending}
              >
                Cancel
              </Button>
            </div>
          </div>
        ) : (
          <Button variant="danger" onClick={() => setConfirmDelete(true)}>
            Delete account
          </Button>
        )}
      </Card>

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
