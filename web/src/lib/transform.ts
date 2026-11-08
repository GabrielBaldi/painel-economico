// Pure data transformations used by the chart, the table and the cards.

import { addMonths } from './dates';
import type { Indicator, Point } from './types';

export interface Series {
  indicator: Indicator;
  points: Point[];
}

/** Keeps the last observation of each month, dated on the 1st of that month. */
export function monthlyLast(points: readonly Point[]): Point[] {
  const byMonth = new Map<string, number>();
  for (const p of points) byMonth.set(`${p.date.slice(0, 7)}-01`, p.value);
  return [...byMonth].map(([date, value]) => ({ date, value }));
}

export interface TableRow {
  date: string;
  a: number | null;
  b: number | null;
}

/**
 * Lines up one or two series by date for the table view, newest first. When
 * a daily series is compared with a monthly one, the daily series is reduced
 * to its last value in each month so every row has both values.
 */
export function toTableRows(a: Series, b: Series | null): TableRow[] {
  const mixed = b !== null && a.indicator.frequency !== b.indicator.frequency;
  const align = (s: Series) =>
    mixed && s.indicator.frequency === 'daily' ? monthlyLast(s.points) : s.points;

  const rows = new Map<string, TableRow>();
  for (const p of align(a)) rows.set(p.date, { date: p.date, a: p.value, b: null });
  if (b) {
    for (const p of align(b)) {
      const row = rows.get(p.date);
      if (row) row.b = p.value;
      else rows.set(p.date, { date: p.date, a: null, b: p.value });
    }
  }
  return [...rows.values()].sort((x, y) => (x.date < y.date ? 1 : x.date > y.date ? -1 : 0));
}

/**
 * Index of the last value in an ascending array that is <= target, or -1.
 * Used to find the value in force on a date (the "as of" value).
 */
export function lastAtOrBefore(sorted: readonly number[], target: number): number {
  let lo = 0;
  let hi = sorted.length - 1;
  let found = -1;
  while (lo <= hi) {
    const mid = (lo + hi) >> 1;
    if ((sorted[mid] ?? Infinity) <= target) {
      found = mid;
      lo = mid + 1;
    } else {
      hi = mid - 1;
    }
  }
  return found;
}

/** Index of the value closest to target in an ascending array, or -1 if empty. */
export function nearestIndex(sorted: readonly number[], target: number): number {
  if (sorted.length === 0) return -1;
  const before = lastAtOrBefore(sorted, target);
  if (before === -1) return 0;
  if (before === sorted.length - 1) return before;
  const after = before + 1;
  return target - (sorted[before] ?? 0) <= (sorted[after] ?? 0) - target ? before : after;
}

export interface YearChange {
  latest: Point;
  base: Point;
  change: number;
  /** Rates change in percentage points; prices in percent. */
  kind: 'points' | 'percent';
}

/**
 * Compares the latest value with the one in force twelve months earlier.
 * Returns null when the series does not reach back that far.
 */
export function yearChange(points: readonly Point[], unit: string): YearChange | null {
  const latest = points.at(-1);
  if (!latest) return null;
  const target = addMonths(latest.date, -12);
  const base = points.findLast((p) => p.date <= target);
  if (!base) return null;
  const kind = unit.startsWith('%') ? 'points' : 'percent';
  const change =
    kind === 'points' ? latest.value - base.value : (latest.value / base.value - 1) * 100;
  return { latest, base, change, kind };
}

export interface Extremes {
  first: Point;
  last: Point;
  min: Point;
  max: Point;
}

/** First, last, lowest and highest points (earliest wins ties), or null if empty. */
export function extremes(points: readonly Point[]): Extremes | null {
  const first = points[0];
  const last = points.at(-1);
  if (!first || !last) return null;
  let min = first;
  let max = first;
  for (const p of points) {
    if (p.value < min.value) min = p;
    if (p.value > max.value) max = p;
  }
  return { first, last, min, max };
}
