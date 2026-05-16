/**
 * Binary search for the index in `arr` whose numeric key is closest to
 * `target`. `arr` must be sorted ascending by `key`. Ties (equidistant
 * neighbours) prefer the lower index for stable behaviour.
 *
 * @param arr Sorted array to search.
 * @param target Value to compare against.
 * @param key Maps an array element to the numeric key being searched.
 * @returns Index of the nearest element, or -1 if `arr` is empty.
 */
export function findNearestIndex<T>(
  arr: readonly T[],
  target: number,
  key: (item: T) => number
): number {
  const n = arr.length
  if (n === 0) return -1
  if (n === 1) return 0

  let lo = 0
  let hi = n - 1
  while (lo < hi) {
    const mid = (lo + hi) >>> 1
    if (key(arr[mid]) < target) lo = mid + 1
    else hi = mid
  }
  // `lo` is the first index with key(arr[lo]) >= target.
  if (lo === 0) return 0
  const distBefore = target - key(arr[lo - 1])
  const distAt = key(arr[lo]) - target
  return distBefore <= distAt ? lo - 1 : lo
}
