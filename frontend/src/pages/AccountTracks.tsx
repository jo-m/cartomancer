import TrackGrid from "../components/TrackGrid"
import useDocumentTitle from "../hooks/useDocumentTitle"

export default function AccountTracks() {
  useDocumentTitle("My Tracks")
  return <TrackGrid mode="user" />
}
