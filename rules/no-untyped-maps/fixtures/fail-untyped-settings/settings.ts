export function loadSettings(raw: string): Record<string, unknown> {
  return JSON.parse(raw) as Record<string, unknown>;
}

export function themeOf(settings: Record<string, unknown>): string {
  return typeof settings["theme"] === "string" ? (settings["theme"] as string) : "light";
}

export function fontSizeOf(settings: Record<string, unknown>): number {
  return typeof settings["fontSize"] === "number" ? (settings["fontSize"] as number) : 14;
}
