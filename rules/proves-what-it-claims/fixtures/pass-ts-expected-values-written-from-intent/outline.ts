export interface Section {
  title: string;
  level: number;
  children: Section[];
}

export function buildOutline(markdown: string): Section[] {
  const roots: Section[] = [];
  const open: Section[] = [];

  for (const line of markdown.split("\n")) {
    const heading = parseHeading(line);
    if (heading === null) {
      continue;
    }

    while (open.length > 0 && open[open.length - 1].level >= heading.level) {
      open.pop();
    }

    const parent = open[open.length - 1];
    if (parent === undefined) {
      roots.push(heading);
    } else {
      parent.children.push(heading);
    }

    open.push(heading);
  }

  return roots;
}

export function renderOutline(sections: Section[]): string {
  return sections.flatMap((section) => renderLines(section, 0)).join("\n");
}

function parseHeading(line: string): Section | null {
  const match = /^(#{1,3}) +(.+?) *$/.exec(line);
  if (match === null) {
    return null;
  }

  return { title: match[2], level: match[1].length, children: [] };
}

function renderLines(section: Section, depth: number): string[] {
  return [
    "  ".repeat(depth) + section.title,
    ...section.children.flatMap((child) => renderLines(child, depth + 1)),
  ];
}
