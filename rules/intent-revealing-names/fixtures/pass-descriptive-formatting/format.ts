export function formatGenerationTimestamp(generationTimestamp: Date): string {
  const isoTimestamp = generationTimestamp.toISOString();
  return isoTimestamp.replace("T", " ").slice(0, 19);
}

export function isOpeningTag(fragment: string): boolean {
  const hasAngleBrackets = fragment.startsWith("<") && fragment.endsWith(">");
  return hasAngleBrackets && !fragment.startsWith("</");
}
