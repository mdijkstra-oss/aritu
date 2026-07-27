export function formatDuration(seconds: number): string {
  const hours = Math.floor(seconds / SECONDS_PER_HOUR);
  const minutes = Math.floor((seconds % SECONDS_PER_HOUR) / SECONDS_PER_MINUTE);
  const remainder = seconds % SECONDS_PER_MINUTE;

  const parts = [`${remainder}s`];
  if (hours > 0 || minutes > 0) {
    parts.unshift(`${minutes}m`);
  }
  if (hours > 0) {
    parts.unshift(`${hours}h`);
  }

  return parts.join(" ");
}

const SECONDS_PER_MINUTE = 60;
const SECONDS_PER_HOUR = 3600;
