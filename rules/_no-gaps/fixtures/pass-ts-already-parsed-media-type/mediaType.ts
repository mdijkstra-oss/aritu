export type MediaType = {
  type: string;
  subtype: string;
  suffix: string | null;
};

export function isJsonPayload(media: MediaType): boolean {
  if (media.type !== "application") {
    return false;
  }

  if (media.subtype === "json") {
    return true;
  }

  return media.suffix === "json";
}
