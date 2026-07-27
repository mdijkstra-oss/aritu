export interface Task {
  title: string;
  done: boolean;
}

export function filterOpen(tasks: Task[]): Task[] {
  return tasks.filter((task) => !task.done);
}

export function getOpenCount(tasks: Task[]): number {
  return filterOpen(tasks).length;
}

export function findNextOpen(tasks: Task[]): Task | undefined {
  return filterOpen(tasks)[0];
}
