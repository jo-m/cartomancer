import { useState } from "react"
import { useQueryClient } from "@tanstack/react-query"
import { $api } from "../api/client"
import { useSession } from "../context/SessionContext"
import Button from "./ui/Button"

/** Formats a date-time string as a short human-readable date. */
function formatCommentDate(iso: string): string {
  return new Date(iso).toLocaleDateString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
  })
}

interface Comment {
  uuid: string
  trackId: string
  user: { uuid: string; name: string }
  body: string
  bodyHtml: string
  deleted: boolean
  createdAt: string
  updatedAt: string
  canEdit: boolean
  canDelete: boolean
}

interface CommentSectionProps {
  trackUUID: string
  isPublicOrOwner: boolean
  onError: (msg: string) => void
}

/** Displays a list of comments with creation, editing, and deletion support. */
export default function CommentSection({
  trackUUID,
  isPublicOrOwner,
  onError,
}: CommentSectionProps) {
  const { user } = useSession()

  const { data, isLoading } = $api.useQuery("get", "/tracks/{uuid}/comments", {
    params: { path: { uuid: trackUUID } },
    enabled: isPublicOrOwner,
  })

  const comments = (data?.comments ?? []) as Comment[]

  if (!isPublicOrOwner) return null

  return (
    <div className="mt-8">
      <h2 className="text-lg font-semibold text-text mb-4">Comments</h2>

      {isLoading && <p className="text-text-muted text-sm">Loading...</p>}

      {!isLoading && comments.length === 0 && (
        <p className="text-text-muted text-sm">No comments yet.</p>
      )}

      <div className="space-y-4">
        {comments.map((comment) => (
          <CommentItem
            key={comment.uuid}
            comment={comment}
            trackUUID={trackUUID}
            onError={onError}
          />
        ))}
      </div>

      {user && <CommentForm trackUUID={trackUUID} onError={onError} />}
    </div>
  )
}

interface CommentItemProps {
  comment: Comment
  trackUUID: string
  onError: (msg: string) => void
}

function CommentItem({ comment, trackUUID, onError }: CommentItemProps) {
  const queryClient = useQueryClient()
  const [editing, setEditing] = useState(false)
  const [editBody, setEditBody] = useState(comment.body)

  const editMutation = $api.useMutation(
    "patch",
    "/tracks/{uuid}/comments/{commentUUID}"
  )
  const deleteMutation = $api.useMutation(
    "delete",
    "/tracks/{uuid}/comments/{commentUUID}"
  )

  async function handleSave() {
    if (!editBody.trim()) return
    try {
      await editMutation.mutateAsync({
        params: { path: { uuid: trackUUID, commentUUID: comment.uuid } },
        body: { body: editBody.trim() },
      })
      setEditing(false)
      await queryClient.invalidateQueries({
        queryKey: ["get", "/tracks/{uuid}/comments"],
      })
    } catch (err) {
      onError((err as Error).message)
    }
  }

  async function handleDelete() {
    try {
      await deleteMutation.mutateAsync({
        params: { path: { uuid: trackUUID, commentUUID: comment.uuid } },
      })
      await queryClient.invalidateQueries({
        queryKey: ["get", "/tracks/{uuid}/comments"],
      })
    } catch (err) {
      onError((err as Error).message)
    }
  }

  if (comment.deleted) {
    return (
      <div className="rounded-lg border border-border bg-surface px-4 py-3">
        <div className="flex items-center gap-2 text-sm text-text-muted">
          <img
            src={`/api/users/${comment.user.uuid}/avatar`}
            alt=""
            className="h-5 w-5 rounded-full"
          />
          <span>{comment.user.name}</span>
          <span>-</span>
          <span>{formatCommentDate(comment.createdAt)}</span>
        </div>
        <p className="mt-2 text-sm text-text-muted italic">
          This comment was deleted.
        </p>
      </div>
    )
  }

  return (
    <div className="rounded-lg border border-border bg-panel px-4 py-3">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2 text-sm text-text-muted">
          <img
            src={`/api/users/${comment.user.uuid}/avatar`}
            alt=""
            className="h-5 w-5 rounded-full"
          />
          <span className="font-medium text-text-secondary">
            {comment.user.name}
          </span>
          <span>-</span>
          <span>{formatCommentDate(comment.createdAt)}</span>
          {comment.updatedAt !== comment.createdAt && (
            <span className="text-xs">(edited)</span>
          )}
        </div>
        {(comment.canEdit || comment.canDelete) && !editing && (
          <div className="flex gap-1">
            {comment.canEdit && (
              <button
                type="button"
                onClick={() => {
                  setEditBody(comment.body)
                  setEditing(true)
                }}
                className="inline-flex min-h-11 cursor-pointer items-center px-2 text-xs text-text-muted hover:text-text-secondary transition-colors"
              >
                Edit
              </button>
            )}
            {comment.canDelete && (
              <button
                type="button"
                onClick={handleDelete}
                disabled={deleteMutation.isPending}
                className="inline-flex min-h-11 cursor-pointer items-center px-2 text-xs text-text-muted hover:text-error transition-colors disabled:opacity-50"
              >
                Delete
              </button>
            )}
          </div>
        )}
      </div>

      {editing ? (
        <div className="mt-2">
          <textarea
            value={editBody}
            onChange={(e) => setEditBody(e.target.value)}
            rows={3}
            className="w-full rounded border border-border bg-surface px-3 py-2 text-sm text-text focus:border-primary focus:outline-none transition-colors"
          />
          <div className="mt-2 flex gap-2">
            <Button
              variant="primary"
              onClick={handleSave}
              disabled={editMutation.isPending || !editBody.trim()}
              className="px-3"
            >
              Save
            </Button>
            <Button
              variant="ghost"
              onClick={() => setEditing(false)}
              className="px-3"
            >
              Cancel
            </Button>
          </div>
        </div>
      ) : (
        <div
          className="prose mt-2 text-sm text-text"
          dangerouslySetInnerHTML={{ __html: comment.bodyHtml }}
        />
      )}
    </div>
  )
}

interface CommentFormProps {
  trackUUID: string
  onError: (msg: string) => void
}

function CommentForm({ trackUUID, onError }: CommentFormProps) {
  const queryClient = useQueryClient()
  const [body, setBody] = useState("")

  const createMutation = $api.useMutation("post", "/tracks/{uuid}/comments")

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!body.trim()) return
    try {
      await createMutation.mutateAsync({
        params: { path: { uuid: trackUUID } },
        body: { body: body.trim() },
      })
      setBody("")
      await queryClient.invalidateQueries({
        queryKey: ["get", "/tracks/{uuid}/comments"],
      })
    } catch (err) {
      onError((err as Error).message)
    }
  }

  return (
    <form onSubmit={handleSubmit} className="mt-4">
      <textarea
        value={body}
        onChange={(e) => setBody(e.target.value)}
        placeholder="Add a comment..."
        rows={3}
        className="w-full rounded border border-border bg-surface px-3 py-2 text-sm text-text placeholder:text-text-muted focus:border-primary focus:outline-none transition-colors"
      />
      <div className="mt-2 flex items-center justify-between">
        <span className="text-xs text-text-muted">
          Supports **bold**, *italic*, and lists.
        </span>
        <Button
          type="submit"
          variant="primary"
          disabled={createMutation.isPending || !body.trim()}
          className="px-3"
        >
          Comment
        </Button>
      </div>
    </form>
  )
}
