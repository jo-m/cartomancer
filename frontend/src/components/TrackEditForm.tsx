import { useState } from "react"
import { useNavigate } from "react-router-dom"
import { useQueryClient } from "@tanstack/react-query"
import { useForm, Controller } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import { $api } from "../api/client"
import {
  SPORT_LABELS,
  SUB_SPORT_LABELS,
  SUB_SPORTS_BY_SPORT,
} from "../lib/sports"
import TagsInput from "./TagsInput"
import Button from "./ui/Button"
import Input from "./ui/Input"
import Select from "./ui/Select"

const editSchema = z.object({
  name: z.string().min(1, "Name is required"),
  public: z.boolean(),
  trackType: z.number().int(),
  sport: z.number().int(),
  subSport: z.number().int(),
  tags: z.array(z.string()),
})

type EditFormValues = z.infer<typeof editSchema>

interface TrackEditData {
  uuid: string
  name: string
  public?: boolean
  trackType: number
  sport: number
  subSport: number
  tags: string[]
}

export interface TrackEditFormProps {
  track: TrackEditData
  onError: (msg: string) => void
  onSuccess: (msg: string) => void
}

/** Edit form for track metadata with delete confirmation. */
export default function TrackEditForm({
  track,
  onError,
  onSuccess,
}: TrackEditFormProps) {
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const [confirmDelete, setConfirmDelete] = useState(false)

  const editMutation = $api.useMutation("patch", "/tracks/{uuid}")
  const deleteMutation = $api.useMutation("delete", "/tracks/{uuid}")

  const {
    register,
    handleSubmit,
    control,
    watch,
    formState: { errors, isSubmitting },
  } = useForm<EditFormValues>({
    resolver: zodResolver(editSchema),
    values: {
      name: track.name,
      public: track.public ?? false,
      trackType: track.trackType,
      sport: track.sport,
      subSport: track.subSport,
      tags: track.tags,
    },
  })

  const watchedSport = watch("sport")

  async function onSubmit(values: EditFormValues) {
    try {
      await editMutation.mutateAsync({
        params: { path: { uuid: track.uuid } },
        body: {
          name: values.name,
          public: values.public,
          trackType: values.trackType,
          sport: values.sport,
          subSport: values.subSport,
          tags: values.tags,
        },
      })
      await queryClient.invalidateQueries({
        queryKey: ["get", "/tracks/{uuid}"],
      })
      onSuccess("Track saved.")
    } catch (err) {
      onError((err as Error).message)
    }
  }

  async function handleDelete() {
    try {
      await deleteMutation.mutateAsync({
        params: { path: { uuid: track.uuid } },
      })
      navigate("/")
    } catch (err) {
      onError((err as Error).message)
      setConfirmDelete(false)
    }
  }

  return (
    <div className="mt-8 border-t border-border pt-6">
      <h2 className="text-sm font-medium uppercase tracking-wide text-text-muted">
        Edit
      </h2>
      <form onSubmit={handleSubmit(onSubmit)} className="mt-4 space-y-4">
        <Input
          label="Name"
          error={errors.name?.message}
          {...register("name")}
        />

        <div className="flex items-center gap-4">
          <div className="flex items-center gap-2">
            <input
              type="checkbox"
              id="track-public"
              {...register("public")}
              className="rounded border-border accent-primary"
            />
            <label
              htmlFor="track-public"
              className="text-sm text-text-secondary"
            >
              Public
            </label>
          </div>
          <div className="flex items-center gap-2">
            <label className="text-sm text-text-secondary">Type</label>
            <Select
              {...register("trackType", { valueAsNumber: true })}
              className="px-2 py-1 text-sm"
            >
              <option value={2}>Recorded</option>
              <option value={1}>Planned</option>
            </Select>
          </div>
        </div>

        <div className="grid grid-cols-2 gap-4">
          <Select label="Sport" {...register("sport", { valueAsNumber: true })}>
            {Object.entries(SPORT_LABELS).map(([k, v]) => (
              <option key={k} value={k}>
                {v}
              </option>
            ))}
          </Select>
          <Select
            label="Sub-sport"
            {...register("subSport", { valueAsNumber: true })}
          >
            {(SUB_SPORTS_BY_SPORT[watchedSport] ?? [0]).map((id) => (
              <option key={id} value={id}>
                {SUB_SPORT_LABELS[id]}
              </option>
            ))}
          </Select>
        </div>

        <div>
          <label className="block text-xs font-medium text-text-secondary">
            Tags
          </label>
          <div className="mt-1">
            <Controller
              name="tags"
              control={control}
              render={({ field }) => (
                <TagsInput
                  value={field.value ?? []}
                  onChange={field.onChange}
                />
              )}
            />
          </div>
        </div>

        <div className="flex items-center gap-3 pt-1">
          <Button
            type="submit"
            variant="secondary"
            disabled={isSubmitting || editMutation.isPending}
          >
            {editMutation.isPending ? "Saving..." : "Save"}
          </Button>

          <div className="ml-auto flex items-center gap-2">
            {confirmDelete ? (
              <>
                <span className="text-sm text-text-secondary">
                  Delete this track?
                </span>
                <Button
                  variant="danger"
                  onClick={handleDelete}
                  disabled={deleteMutation.isPending}
                >
                  {deleteMutation.isPending ? "Deleting..." : "Confirm"}
                </Button>
                <Button
                  variant="secondary"
                  onClick={() => setConfirmDelete(false)}
                >
                  Cancel
                </Button>
              </>
            ) : (
              <Button
                variant="secondary"
                onClick={() => setConfirmDelete(true)}
              >
                Delete
              </Button>
            )}
          </div>
        </div>
      </form>
    </div>
  )
}
