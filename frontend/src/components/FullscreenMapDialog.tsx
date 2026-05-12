import { Fragment } from "react"
import {
  Dialog,
  DialogPanel,
  Transition,
  TransitionChild,
} from "@headlessui/react"
import { XMarkIcon } from "@heroicons/react/24/outline"
import TrackMap from "./TrackMap"
import type { RoadClosure } from "../types/map"
import { useHoverStore } from "../hooks/useHoverSync"
import MapHoverOverlay from "./MapHoverOverlay"
import type { MapLayer } from "../lib/mapLayer"

export interface FullscreenMapDialogProps {
  open: boolean
  onClose: () => void
  trackPoints: { lat: number; lon: number; ele: number; d: number }[]
  hoverStore: ReturnType<typeof useHoverStore>
  color: string
  closures?: RoadClosure[]
  forecastTimes?: number[]
  layer: MapLayer
}

/** Fullscreen map dialog with transition animations. */
export default function FullscreenMapDialog({
  open,
  onClose,
  trackPoints,
  hoverStore,
  color,
  closures,
  forecastTimes,
  layer,
}: FullscreenMapDialogProps) {
  return (
    <Transition show={open} as={Fragment}>
      <Dialog onClose={onClose} className="relative z-50">
        <TransitionChild
          as={Fragment}
          enter="ease-out duration-200"
          enterFrom="opacity-0"
          enterTo="opacity-100"
          leave="ease-in duration-150"
          leaveFrom="opacity-100"
          leaveTo="opacity-0"
        >
          <div className="fixed inset-0 bg-overlay" />
        </TransitionChild>
        <TransitionChild
          as={Fragment}
          enter="ease-out duration-200"
          enterFrom="opacity-0 scale-95"
          enterTo="opacity-100 scale-100"
          leave="ease-in duration-150"
          leaveFrom="opacity-100 scale-100"
          leaveTo="opacity-0 scale-95"
        >
          <DialogPanel className="fixed inset-0 flex flex-col bg-panel">
            <button
              type="button"
              onClick={onClose}
              className="absolute top-3 right-3 z-10 cursor-pointer rounded bg-panel/90 p-1.5 text-text-secondary shadow-sm hover:bg-panel hover:text-text transition-colors"
              aria-label="Close fullscreen"
            >
              <XMarkIcon className="h-6 w-6" />
            </button>
            <div className="relative h-full w-full">
              <TrackMap
                points={trackPoints}
                hoverStore={hoverStore}
                color={color}
                className="h-full w-full"
                closures={closures}
                layer={layer}
              />
              <MapHoverOverlay
                hoverStore={hoverStore}
                trackPoints={trackPoints}
                forecastTimes={forecastTimes}
              />
            </div>
          </DialogPanel>
        </TransitionChild>
      </Dialog>
    </Transition>
  )
}
