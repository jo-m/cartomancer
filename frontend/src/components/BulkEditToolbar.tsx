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
import SectionHeading from "./ui/SectionHeading"

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

  const linkBtn =
    "inline-flex min-h-11 cursor-pointer items-center px-2 text-xs text-text-muted hover:text-text-secondary disabled:opacity-50 transition-colors"

  return (
    <div
      data-bulk-toolbar
      className="mb-4 rounded-lg border border-primary bg-panel px-4 py-3"
    >
      <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
        <span className="text-sm font-medium text-text">
          {selected.size} selected
        </span>
        <div className="flex flex-wrap items-center gap-1">
          <button onClick={() => onSelectAll()} className={linkBtn}>
            Select all on page
          </button>
          <button onClick={clearAndReset} className={linkBtn}>
            Clear
          </button>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <div className="flex flex-col gap-1">
          <SectionHeading>Visibility</SectionHeading>
          <div className="flex flex-wrap gap-2">
            <Button
              variant="secondary"
              onClick={() => bulkSetVisibility(true)}
              disabled={bulkEditMutation.isPending}
              className="px-3 text-xs"
            >
              Set public
            </Button>
            <Button
              variant="secondary"
              onClick={() => bulkSetVisibility(false)}
              disabled={bulkEditMutation.isPending}
              className="px-3 text-xs"
            >
              Set private
            </Button>
          </div>
        </div>

        <div className="flex flex-col gap-1">
          <SectionHeading>Type</SectionHeading>
          <div className="flex flex-wrap gap-2">
            <Button
              variant="secondary"
              onClick={() => bulkSetTrackType(2)}
              disabled={bulkEditMutation.isPending}
              className="px-3 text-xs"
            >
              Recorded
            </Button>
            <Button
              variant="secondary"
              onClick={() => bulkSetTrackType(1)}
              disabled={bulkEditMutation.isPending}
              className="px-3 text-xs"
            >
              Planned
            </Button>
          </div>
        </div>

        <div className="flex flex-col gap-1">
          <SectionHeading>Sport</SectionHeading>
          <div className="flex flex-wrap items-center gap-2">
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
            <Button
              variant="secondary"
              onClick={bulkApplySport}
              disabled={!bulkSport || bulkEditMutation.isPending}
              className="px-3 text-xs"
            >
              Apply
            </Button>
          </div>
        </div>

        <div className="flex flex-col gap-1">
          <SectionHeading>Tags</SectionHeading>
          <div className="flex flex-wrap items-center gap-2">
            <div className="min-w-0 flex-1 basis-full sm:basis-auto">
              <TagsInput value={bulkTags} onChange={setBulkTags} />
            </div>
            <Button
              variant="secondary"
              onClick={bulkApplyTags}
              disabled={bulkTags.length === 0 || bulkEditMutation.isPending}
              className="px-3 text-xs"
            >
              Apply
            </Button>
          </div>
        </div>
      </div>

      <div className="mt-3 flex flex-wrap items-center gap-2 border-t border-border pt-3">
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
            <button onClick={() => setConfirmDelete(false)} className={linkBtn}>
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
