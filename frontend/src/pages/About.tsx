import { $api } from "../api/client"

/** About page showing build version and dependency information. */
export default function About() {
  const { data, isLoading, error } = $api.useQuery("get", "/version")

  return (
    <div className="mx-auto max-w-5xl px-4 py-8">
      <h1 className="mb-6 text-2xl font-semibold text-gray-900">About</h1>

      {isLoading && <p className="text-gray-500">Loading...</p>}
      {error && (
        <p className="text-red-600">
          Failed to load version info: {(error as unknown as Error).message}
        </p>
      )}

      {data && (
        <div className="space-y-6">
          <table className="w-full text-sm">
            <tbody className="divide-y divide-gray-100">
              <tr>
                <td className="py-2 pr-4 font-medium text-gray-500 w-32">
                  Go version
                </td>
                <td className="py-2 font-mono text-gray-900">
                  {data.goVersion}
                </td>
              </tr>
              <tr>
                <td className="py-2 pr-4 font-medium text-gray-500">Module</td>
                <td className="py-2 font-mono text-gray-900">
                  <a
                    href={`https://pkg.go.dev/${data.path}`}
                    target="_blank"
                    rel="noreferrer"
                    className="underline hover:text-gray-600"
                  >
                    {data.path}
                  </a>
                </td>
              </tr>
              <tr>
                <td className="py-2 pr-4 font-medium text-gray-500">Version</td>
                <td className="py-2 font-mono text-gray-900">
                  {data.version || "(devel)"}
                </td>
              </tr>
            </tbody>
          </table>

          <div>
            <h2 className="mb-2 text-lg font-medium text-gray-800">
              Data Sources
            </h2>
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-gray-200 text-left text-xs font-medium uppercase text-gray-500">
                  <th className="pb-2 pr-4">Used for</th>
                  <th className="pb-2 pr-4">Title</th>
                  <th className="pb-2 pr-4">Author</th>
                  <th className="pb-2 pr-4">Source</th>
                  <th className="pb-2">License</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {data.attributions.map((attr) => (
                  <tr key={attr.title}>
                    <td className="py-1 pr-4 text-gray-700">{attr.what}</td>
                    <td className="py-1 pr-4 text-gray-700">{attr.title}</td>
                    <td className="py-1 pr-4 text-gray-700">{attr.author}</td>
                    <td className="py-1 pr-4 text-gray-700">
                      <a
                        href={attr.source}
                        target="_blank"
                        rel="noreferrer"
                        className="underline hover:text-gray-500"
                      >
                        {attr.source}
                      </a>
                    </td>
                    <td className="py-1 text-gray-700">
                      <a
                        href={attr.licenseUrl}
                        target="_blank"
                        rel="noreferrer"
                        className="underline hover:text-gray-500"
                      >
                        {attr.license}
                      </a>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          <div>
            <h2 className="mb-2 text-lg font-medium text-gray-800">
              Dependencies
            </h2>
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-gray-200 text-left text-xs font-medium uppercase text-gray-500">
                  <th className="pb-2 pr-4">Module</th>
                  <th className="pb-2">Version</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {data.deps.map((dep) => (
                  <tr key={dep.path}>
                    <td className="py-1 pr-4 font-mono text-gray-700">
                      <a
                        href={`https://pkg.go.dev/${dep.path}`}
                        target="_blank"
                        rel="noreferrer"
                        className="underline hover:text-gray-500"
                      >
                        {dep.path}
                      </a>
                    </td>
                    <td className="py-1 font-mono text-gray-500">
                      {dep.version}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  )
}
