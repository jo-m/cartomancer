import { $api } from "../api/client"
import useDocumentTitle from "../hooks/useDocumentTitle"
import PageContainer from "../components/ui/PageContainer"
import Card from "../components/ui/Card"

/** About page showing build version and dependency information. */
export default function About() {
  useDocumentTitle("About")
  const { data, isLoading, error } = $api.useQuery("get", "/version")

  return (
    <PageContainer>
      <h1 className="mb-6 text-2xl font-semibold text-text">About</h1>

      {isLoading && <p className="text-text-muted">Loading...</p>}
      {error && (
        <p role="alert" className="text-error">
          Failed to load version info: {error.message}
        </p>
      )}

      {data && (
        <div className="space-y-6">
          <Card className="overflow-hidden">
            <table className="w-full text-sm">
              <tbody className="divide-y divide-border">
                <tr>
                  <td className="py-2 px-4 pr-4 font-medium text-text-muted w-32">
                    Source code
                  </td>
                  <td className="py-2 px-4 font-mono text-text">
                    <a
                      href="https://github.com/jo-m/cartomancer"
                      target="_blank"
                      rel="noreferrer"
                      className="underline hover:text-text-secondary transition-colors"
                    >
                      github.com/jo-m/cartomancer
                    </a>
                  </td>
                </tr>
                <tr>
                  <td className="py-2 px-4 pr-4 font-medium text-text-muted">
                    Module
                  </td>
                  <td className="py-2 px-4 font-mono text-text">
                    <a
                      href={`https://${data.path}`}
                      target="_blank"
                      rel="noreferrer"
                      className="underline hover:text-text-secondary transition-colors"
                    >
                      {data.path}
                    </a>
                  </td>
                </tr>
                <tr>
                  <td className="py-2 px-4 pr-4 font-medium text-text-muted w-32">
                    Go version
                  </td>
                  <td className="py-2 px-4 font-mono text-text">
                    {data.goVersion}
                  </td>
                </tr>
                <tr>
                  <td className="py-2 px-4 pr-4 font-medium text-text-muted">
                    Version
                  </td>
                  <td className="py-2 px-4 font-mono text-text">
                    {data.version || "(devel)"}
                  </td>
                </tr>
              </tbody>
            </table>
          </Card>

          <div>
            <h2 className="mb-2 text-lg font-medium text-text">Data Sources</h2>
            <Card className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border text-left text-xs font-medium uppercase text-text-muted">
                    <th className="px-4 pb-2 pt-3 pr-4">Used for</th>
                    <th className="px-4 pb-2 pt-3 pr-4">Title</th>
                    <th className="px-4 pb-2 pt-3 pr-4">Author</th>
                    <th className="px-4 pb-2 pt-3 pr-4">Source</th>
                    <th className="px-4 pb-2 pt-3">License</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-border">
                  {data.attributions.map((attr) => (
                    <tr key={attr.title}>
                      <td className="py-1 px-4 pr-4 text-text-secondary">
                        {attr.what}
                      </td>
                      <td className="py-1 px-4 pr-4 text-text-secondary">
                        {attr.title}
                      </td>
                      <td className="py-1 px-4 pr-4 text-text-secondary">
                        {attr.author}
                      </td>
                      <td className="py-1 px-4 pr-4 text-text-secondary">
                        <a
                          href={attr.source}
                          target="_blank"
                          rel="noreferrer"
                          className="underline hover:text-text-muted transition-colors"
                        >
                          {attr.source}
                        </a>
                      </td>
                      <td className="py-1 px-4 text-text-secondary">
                        <a
                          href={attr.licenseUrl}
                          target="_blank"
                          rel="noreferrer"
                          className="underline hover:text-text-muted transition-colors"
                        >
                          {attr.license}
                        </a>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </Card>
          </div>

          <div>
            <h2 className="mb-2 text-lg font-medium text-text">Dependencies</h2>
            <Card className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border text-left text-xs font-medium uppercase text-text-muted">
                    <th className="px-4 pb-2 pt-3 pr-4">Module</th>
                    <th className="px-4 pb-2 pt-3">Version</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-border">
                  {data.deps.map((dep) => (
                    <tr key={dep.path}>
                      <td className="py-1 px-4 pr-4 font-mono text-text-secondary">
                        <a
                          href={`https://pkg.go.dev/${dep.path}`}
                          target="_blank"
                          rel="noreferrer"
                          className="underline hover:text-text-muted transition-colors"
                        >
                          {dep.path}
                        </a>
                      </td>
                      <td className="py-1 px-4 font-mono text-text-muted">
                        {dep.version}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </Card>
          </div>
        </div>
      )}
    </PageContainer>
  )
}
