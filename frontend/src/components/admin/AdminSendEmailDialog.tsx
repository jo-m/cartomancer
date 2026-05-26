import { Fragment } from "react"
import {
  Dialog,
  DialogPanel,
  DialogTitle,
  Transition,
  TransitionChild,
} from "@headlessui/react"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import { fetchClient } from "../../api/client"
import Button from "../ui/Button"
import Input from "../ui/Input"

const SUBJECT_MAX_LEN = 256
const BODY_MAX_LEN = 16384

const sendEmailSchema = z.object({
  subject: z
    .string()
    .min(1, "Required")
    .max(SUBJECT_MAX_LEN, `Max ${SUBJECT_MAX_LEN} characters`)
    .refine((s) => !/[\r\n]/.test(s), "Must not contain line breaks"),
  body: z
    .string()
    .min(1, "Required")
    .max(BODY_MAX_LEN, `Max ${BODY_MAX_LEN} characters`),
})

type SendEmailFormData = z.infer<typeof sendEmailSchema>

export interface AdminSendEmailDialogProps {
  /** UUID of the recipient user. Dialog is open iff this is non-null. */
  userUuid: string | null
  /** Email address shown for confirmation in the dialog header. */
  userEmail: string | null
  /** Closes the dialog without sending. */
  onClose: () => void
  /** Called after the email has been queued successfully. */
  onSent: () => void
  /** Called when the server returns an error. */
  onError: (message: string) => void
}

/**
 * Modal form for an admin to send an arbitrary plain-text email to a single user.
 * The dialog mounts only when [userUuid] is non-null and resets its form on each
 * open so the previous recipient's draft does not leak across invocations.
 */
export default function AdminSendEmailDialog({
  userUuid,
  userEmail,
  onClose,
  onSent,
  onError,
}: AdminSendEmailDialogProps) {
  const {
    register,
    handleSubmit,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<SendEmailFormData>({
    resolver: zodResolver(sendEmailSchema),
    defaultValues: { subject: "", body: "" },
  })

  const open = userUuid !== null

  async function onSubmit(data: SendEmailFormData) {
    if (!userUuid) return
    try {
      await fetchClient.POST("/admin/users/{uuid}/send-email", {
        params: { path: { uuid: userUuid } },
        body: data,
      })
      reset({ subject: "", body: "" })
      onSent()
    } catch (err) {
      onError((err as Error).message)
    }
  }

  function handleClose() {
    if (isSubmitting) return
    reset({ subject: "", body: "" })
    onClose()
  }

  return (
    <Transition show={open} as={Fragment}>
      <Dialog onClose={handleClose} className="relative z-50">
        <TransitionChild
          as={Fragment}
          enter="ease-out duration-150"
          enterFrom="opacity-0"
          enterTo="opacity-100"
          leave="ease-in duration-100"
          leaveFrom="opacity-100"
          leaveTo="opacity-0"
        >
          <div className="fixed inset-0 bg-overlay" />
        </TransitionChild>
        <div className="fixed inset-0 flex items-start justify-center overflow-y-auto p-4 sm:items-center">
          <TransitionChild
            as={Fragment}
            enter="ease-out duration-150"
            enterFrom="opacity-0 scale-95"
            enterTo="opacity-100 scale-100"
            leave="ease-in duration-100"
            leaveFrom="opacity-100 scale-100"
            leaveTo="opacity-0 scale-95"
          >
            <DialogPanel className="w-full max-w-xl rounded border border-border bg-panel p-5 shadow-lg">
              <DialogTitle className="mb-1 text-base font-medium text-text">
                Send email
              </DialogTitle>
              {userEmail && (
                <p className="mb-4 text-sm text-text-muted">
                  To: <span className="text-text-secondary">{userEmail}</span>
                </p>
              )}
              <form
                onSubmit={handleSubmit(onSubmit)}
                className="flex flex-col gap-3"
              >
                <Input
                  type="text"
                  label="Subject"
                  placeholder="Subject"
                  autoComplete="off"
                  maxLength={SUBJECT_MAX_LEN}
                  {...register("subject")}
                  error={errors.subject?.message}
                />
                <div>
                  <label
                    htmlFor="admin-send-email-body"
                    className="mb-1 block text-sm font-medium text-text-secondary"
                  >
                    Body
                  </label>
                  <textarea
                    id="admin-send-email-body"
                    rows={10}
                    maxLength={BODY_MAX_LEN}
                    aria-invalid={errors.body ? "true" : undefined}
                    aria-describedby={
                      errors.body ? "admin-send-email-body-error" : undefined
                    }
                    className={`min-h-32 w-full rounded border border-border bg-panel px-3 py-2 font-mono text-sm text-text placeholder-text-muted transition-colors focus:border-primary focus:outline-none ${errors.body ? "border-error" : ""}`}
                    {...register("body")}
                  />
                  {errors.body?.message && (
                    <p
                      id="admin-send-email-body-error"
                      role="alert"
                      className="mt-1 text-sm text-error"
                    >
                      {errors.body.message}
                    </p>
                  )}
                </div>
                <div className="flex flex-wrap justify-end gap-2">
                  <Button
                    type="button"
                    variant="secondary"
                    onClick={handleClose}
                    disabled={isSubmitting}
                  >
                    Cancel
                  </Button>
                  <Button
                    type="submit"
                    variant="primary"
                    disabled={isSubmitting}
                  >
                    {isSubmitting ? "Sending..." : "Send"}
                  </Button>
                </div>
              </form>
            </DialogPanel>
          </TransitionChild>
        </div>
      </Dialog>
    </Transition>
  )
}
