export function areaOf(
  shape:
    | { kind: "circle"; radius: number }
    | { kind: "square"; size: number }
    | { kind: "triangle"; base: number; height: number },
): number {
  switch (shape.kind) {
    case "circle":
      return Math.PI * shape.radius ** 2;
    case "square":
      return shape.size ** 2;
    default:
      return 0;
  }
}
