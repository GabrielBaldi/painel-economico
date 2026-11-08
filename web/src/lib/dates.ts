// Calendar helpers on ISO dates (YYYY-MM-DD). Working with plain strings and
// UTC timestamps avoids time-zone shifts when a date is parsed or printed.

const ISO_DATE = /^(\d{4})-(\d{2})-(\d{2})$/;

/** Returns true for a real calendar date in YYYY-MM-DD form. */
export function isISODate(value: string): boolean {
  const match = ISO_DATE.exec(value);
  if (!match) return false;
  const [, y, m, d] = match.map(Number) as [number, number, number, number];
  const date = new Date(Date.UTC(y, m - 1, d));
  return date.getUTCFullYear() === y && date.getUTCMonth() === m - 1 && date.getUTCDate() === d;
}

/** Milliseconds since the epoch at UTC midnight of an ISO date. */
export function toTime(iso: string): number {
  return Date.parse(`${iso}T00:00:00Z`);
}

/** The ISO date of a UTC timestamp. */
export function fromTime(time: number): string {
  return new Date(time).toISOString().slice(0, 10);
}

/** The local calendar date of `now` as an ISO date. */
export function localISODate(now: Date): string {
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())}`;
}

/**
 * Moves an ISO date by whole months, clamping the day to the target month
 * (2024-02-29 minus 12 months is 2023-02-28).
 */
export function addMonths(iso: string, months: number): string {
  const [y, m, d] = iso.split('-').map(Number) as [number, number, number];
  const target = new Date(Date.UTC(y, m - 1 + months, 1));
  const lastDay = new Date(
    Date.UTC(target.getUTCFullYear(), target.getUTCMonth() + 1, 0),
  ).getUTCDate();
  target.setUTCDate(Math.min(d, lastDay));
  return fromTime(target.getTime());
}
