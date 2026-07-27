export interface Entry {
  title: string;
  score: number;
}

/** Matches are returned by descending score, ties broken alphabetically by title. */
export function rankMatches(entries: Entry[], query: string): Entry[] {
  const needle = query.toLowerCase();
  const matches = entries.filter((entry) => entry.title.toLowerCase().includes(needle));

  return matches.sort(byScoreThenTitle);
}

function byScoreThenTitle(one: Entry, other: Entry): number {
  return other.score - one.score || one.title.localeCompare(other.title);
}
