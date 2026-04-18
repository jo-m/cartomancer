import TrackGrid from "../components/TrackGrid"
import useDocumentTitle from "../hooks/useDocumentTitle"

export default function Tracks() {
  useDocumentTitle("Tracks")
  return <TrackGrid mode="public" />
}
