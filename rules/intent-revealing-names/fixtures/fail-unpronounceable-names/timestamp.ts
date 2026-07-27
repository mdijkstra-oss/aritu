export function genymdhms(moment: Date): string {
  const ymd = `${moment.getFullYear()}-${moment.getMonth() + 1}-${moment.getDate()}`;
  const hms = `${moment.getHours()}:${moment.getMinutes()}:${moment.getSeconds()}`;
  return `${ymd} ${hms}`;
}
