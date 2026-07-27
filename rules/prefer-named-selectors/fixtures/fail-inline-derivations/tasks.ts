export interface Task {
  title: string;
  done: boolean;
}

export function openCount(tasks: Task[]): number {
  return tasks.filter((task) => !task.done).length;
}

export function openTitles(tasks: Task[]): string[] {
  return tasks.filter((task) => !task.done).map((task) => task.title);
}

export function nextOpen(tasks: Task[]): Task | undefined {
  return tasks.filter((task) => !task.done)[0];
}
