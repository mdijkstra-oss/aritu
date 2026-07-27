import { describe, it, expect } from "vitest";

import { rankMatches, type Entry } from "./search";

describe("rankMatches", () => {
  it("puts the highest scoring match first and breaks ties alphabetically", () => {
    const catalogue: Entry[] = [
      { title: "Pencil case", score: 10 },
      { title: "Notebook", score: 9 },
      { title: "Notepad", score: 5 },
      { title: "Notecards", score: 5 },
      { title: "Sticky notes", score: 2 },
    ];

    const titles = rankMatches(catalogue, "note").map((entry) => entry.title);

    expect(titles).toEqual(["Notebook", "Notecards", "Notepad", "Sticky notes"]);
  });
});
