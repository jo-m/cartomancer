export const SPORT_LABELS: Record<number, string> = {
  0: "Unknown",
  1: "Running",
  2: "Cycling",
}

export const SUB_SPORT_LABELS: Record<number, string> = {
  0: "Unknown",
  1: "Outdoor",
  2: "Treadmill",
  3: "Road",
  4: "Spinning",
  5: "Indoor Cycling",
  6: "Mountain",
  7: "Gravel",
  8: "Commuting",
}

/** Valid sub-sport IDs for each sport ID. */
export const SUB_SPORTS_BY_SPORT: Record<number, number[]> = {
  0: [0],
  1: [0, 1, 2],
  2: [0, 3, 4, 5, 6, 7, 8],
}
