import { useState } from "react"
import { $api, fetchClient } from "../api/client"
import { useQueryClient } from "@tanstack/react-query"
import {
  SPORT_LABELS,
  SUB_SPORT_LABELS,
  SUB_SPORTS_BY_SPORT,
} from "../lib/sports"
import TagsInput from "./TagsInput"
import Button from "./ui/Button"
import Select from "./ui/Select"

export interface BulkEditToolbarProps {
  selected: Set<string>
  onSelectAll: () => void
  onClearSelection: () => void
  onError: (e: unknown) => void
}

/** Toolbar for bulk editing/deleting selected tracks. */
export default function BulkEditToolbar({
  selected,
  onSelectAll,
  onClearSelection,
  onError,
}: BulkEditToolbarProps) {
  const queryClient = useQueryClient()
  const [bulkSport, setBulkSport] = useState("")
  const [bulkSubSport, setBulkSubSport] = useState("")
  const [bulkTags, setBulkTags] = useState<string[]>([])
  const [confirmDelete, setConfirmDelete] = useState(false)

  const bulkEditMutation = $api.useMutation("patch", "/tracks")
  const selectedUuids = [...selected]

  function clearAndReset() {
    onClearSelection()
    setConfirmDelete(false)
    setBulkSport("")
    setBulkSubSport("")
    setBulkTags([])
  }

  function bulkSetVisibility(isPublic: boolean) {
    if (selectedUuids.length === 0) return
    bulkEditMutation.mutate(
      { body: { uuids: selectedUuids, public: isPublic } },
      {
        onSuccess: () => {
          void queryClient.invalidateQueries({ queryKey: ["get", "/tracks"] })
        },
        onError,
      }
    )
  }

  function bulkSetTrackType(trackType: number) {
    if (selectedUuids.length === 0) return
    bulkEditMutation.mutate(
      { body: { uuids: selectedUuids, trackType } },
      {
        onSuccess: () => {
          void queryClient.invalidateQueries({ queryKey: ["get", "/tracks"] })
        },
        onError,
      }
    )
  }

  function bulkApplySport() {
    if (selectedUuids.length === 0 || !bulkSport) return
    const body: Parameters<typeof bulkEditMutation.mutate>[0]["body"] = {
      uuids: selectedUuids,
      sport: parseInt(bulkSport),
    }
    if (bulkSubSport !== "") body.subSport = parseInt(bulkSubSport)
    bulkEditMutation.mutate(
      { body },
      {
        onSuccess: () => {
          void queryClient.invalidateQueries({ queryKey: ["get", "/tracks"] })
        },
        onError,
      }
    )
  }

  function bulkApplyTags() {
    if (selectedUuids.length === 0 || bulkTags.length === 0) return
    bulkEditMutation.mutate(
      { body: { uuids: selectedUuids, tags: bulkTags } },
      {
        onSuccess: () => {
          void queryClient.invalidateQueries({ queryKey: ["get", "/tracks"] })
        },
        onError,
      }
    )
  }

  async function bulkDelete() {
    if (selectedUuids.length === 0) return
    try {
      await fetchClient.POST("/tracks/bulk-delete", {
        body: { uuids: selectedUuids },
      })
      clearAndReset()
      await queryClient.invalidateQueries({ queryKey: ["get", "/tracks"] })
      await queryClient.invalidateQueries({
        queryKey: ["get", "/tracks/statistics"],
      })
    } catch (e) {
      onError(e)
    }
  }

  return (
    <div
      data-bulk-toolbar
      className="mb-4 rounded-lg border border-primary bg-panel px-4 py-3"
    >
      <div className="flex flex-wrap items-center gap-2">
        <span className="text-sm font-medium text-text">
          {selected.size} selected
        </span>
        <button
          onClick={() => onSelectAll()}
          className="inline-flex min-h-11 cursor-pointer items-center px-2 text-xs text-text-muted hover:text-text-secondary transition-colors"
        >
          Select all on page
        </button>
        <button
          onClick={clearAndReset}
          className="inline-flex min-h-11 cursor-pointer items-center px-2 text-xs text-text-muted hover:text-text-secondary transition-colors"
        >
          Clear
        </button>
        <span className="text-border">|</span>
        <button
          onClick={() => bulkSetVisibility(true)}
          disabled={bulkEditMutation.isPending}
          className="inline-flex min-h-11 cursor-pointer items-center px-2 text-xs text-text-muted hover:text-text-secondary disabled:opacity-50 transition-colors"
        >
          Set public
        </button>
        <button
          onClick={() => bulkSetVisibility(false)}
          disabled={bulkEditMutation.isPending}
          className="inline-flex min-h-11 cursor-pointer items-center px-2 text-xs text-text-muted hover:text-text-secondary disabled:opacity-50 transition-colors"
        >
          Set private
        </button>
        <span className="text-border">|</span>
        <button
          onClick={() => bulkSetTrackType(2)}
          disabled={bulkEditMutation.isPending}
          className="inline-flex min-h-11 cursor-pointer items-center px-2 text-xs text-text-muted hover:text-text-secondary disabled:opacity-50 transition-colors"
        >
          Set recorded
        </button>
        <button
          onClick={() => bulkSetTrackType(1)}
          disabled={bulkEditMutation.isPending}
          className="inline-flex min-h-11 cursor-pointer items-center px-2 text-xs text-text-muted hover:text-text-secondary disabled:opacity-50 transition-colors"
        >
          Set planned
        </button>
        <span className="text-border">|</span>
        <Select
          value={bulkSport}
          onChange={(e) => {
            setBulkSport(e.target.value)
            setBulkSubSport("")
          }}
          className="px-2 py-1"
        >
          <option value="">Sport...</option>
          {Object.entries(SPORT_LABELS).map(([id, label]) => (
            <option key={id} value={id}>
              {label}
            </option>
          ))}
        </Select>
        {bulkSport !== "" && (
          <Select
            value={bulkSubSport}
            onChange={(e) => setBulkSubSport(e.target.value)}
            className="px-2 py-1"
          >
            <option value="">Sub-sport...</option>
            {(SUB_SPORTS_BY_SPORT[parseInt(bulkSport)] ?? []).map((id) => (
              <option key={id} value={String(id)}>
                {SUB_SPORT_LABELS[id]}
              </option>
            ))}
          </Select>
        )}
        <button
          onClick={bulkApplySport}
          disabled={!bulkSport || bulkEditMutation.isPending}
          className="inline-flex min-h-11 cursor-pointer items-center px-2 text-xs text-text-muted hover:text-text-secondary disabled:opacity-50 transition-colors"
        >
          Set sport
        </button>
        <span className="text-border">|</span>
        <div className="flex min-w-48 flex-1 items-center gap-2">
          <TagsInput value={bulkTags} onChange={setBulkTags} />
          <button
            onClick={bulkApplyTags}
            disabled={bulkTags.length === 0 || bulkEditMutation.isPending}
            className="inline-flex min-h-11 shrink-0 cursor-pointer items-center px-2 text-xs text-text-muted hover:text-text-secondary disabled:opacity-50 transition-colors"
          >
            Set tags
          </button>
        </div>
      </div>
      <div className="mt-2 flex items-center gap-2">
        {confirmDelete ? (
          <>
            <span className="text-xs text-error">
              Delete {selected.size} track
              {selected.size > 1 ? "s" : ""}?
            </span>
            <Button
              variant="danger"
              onClick={() => void bulkDelete()}
              className="px-3"
            >
              Confirm delete
            </Button>
            <button
              onClick={() => setConfirmDelete(false)}
              className="inline-flex min-h-11 cursor-pointer items-center px-2 text-xs text-text-muted hover:text-text-secondary transition-colors"
            >
              Cancel
            </button>
          </>
        ) : (
          <Button
            variant="danger"
            onClick={() => setConfirmDelete(true)}
            className="px-3"
          >
            Delete selected
          </Button>
        )}
      </div>
    </div>
  )
}
