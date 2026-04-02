import helpHtml from "../assets/help.md"
import PageContainer from "../components/ui/PageContainer"

/** Help page rendered from a Markdown source file. */
export default function Help() {
  return (
    <PageContainer>
      <div className="prose" dangerouslySetInnerHTML={{ __html: helpHtml }} />
    </PageContainer>
  )
}
