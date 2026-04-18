import helpHtml from "../assets/help.md"
import useDocumentTitle from "../hooks/useDocumentTitle"
import PageContainer from "../components/ui/PageContainer"

/** Help page rendered from a Markdown source file. */
export default function Help() {
  useDocumentTitle("Help")
  return (
    <PageContainer>
      <div className="prose" dangerouslySetInnerHTML={{ __html: helpHtml }} />
    </PageContainer>
  )
}
