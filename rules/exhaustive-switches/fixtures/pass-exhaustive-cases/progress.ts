export function labelFor(state: "queued" | "running" | "done"): string {
  switch (state) {
    case "queued":
      return "Waiting to start";
    case "running":
      return "In progress";
    case "done":
      return "Finished";
  }
}

export function countLabel(count: number): string {
  return count === 1 ? "1 item" : `${count} items`;
}
