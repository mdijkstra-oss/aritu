export interface Point {
  x: number;
  y: number;
}

export function distanceBetween(start: Point, end: Point): number {
  return Math.hypot(end.x - start.x, end.y - start.y);
}

export function midpointOf(start: Point, end: Point): Point {
  return { x: (start.x + end.x) / 2, y: (start.y + end.y) / 2 };
}
