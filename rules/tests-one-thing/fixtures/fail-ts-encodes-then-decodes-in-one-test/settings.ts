export type Settings = {
  theme: string;
  fontSize: number;
  autosave: boolean;
};

export function encodeSettings(settings: Settings): string {
  return [
    `theme=${settings.theme}`,
    `fontSize=${settings.fontSize}`,
    `autosave=${settings.autosave}`,
  ].join(";");
}

export function decodeSettings(raw: string): Settings {
  const fields = new Map(raw.split(";").map(toField));

  return {
    theme: fields.get("theme") ?? "",
    fontSize: Number(fields.get("fontSize")),
    autosave: fields.get("autosave") === "true",
  };
}

function toField(part: string): [string, string] {
  const separator = part.indexOf("=");

  return [part.slice(0, separator), part.slice(separator + 1)];
}
