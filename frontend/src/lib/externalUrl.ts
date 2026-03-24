/** Wraps an external URL through the /leaving interstitial page. */
export function externalUrl(url: string): string {
  return `/leaving?url=${encodeURIComponent(url)}`
}
