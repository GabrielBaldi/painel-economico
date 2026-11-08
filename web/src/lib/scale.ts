// Scales and axis ticks for the hand-rolled SVG chart.

import { fromTime } from './dates';
import { formatMonthShort } from './format';

/** Maps a numeric domain onto a pixel range, linearly. */
export function linearScale(
  [d0, d1]: readonly [number, number],
  [r0, r1]: readonly [number, number],
): (value: number) => number {
  const span = d1 - d0 || 1;
  return (value) => r0 + ((value - d0) / span) * (r1 - r0);
}

/** How many decimals a tick step needs: 0.25 → 2, 2.5 → 1, 5 → 0. */
export function decimalsOf(step: number): number {
  const [, decimals = ''] = String(Number(step.toPrecision(12))).split('.');
  return decimals.length;
}

/** A "nice" step (1, 2, 2.5 or 5 times a power of ten) for about `count` intervals. */
function niceStep(span: number, count: number): number {
  const raw = span / count;
  const power = 10 ** Math.floor(Math.log10(raw));
  const fraction = raw / power;
  const nice =
    fraction <= 1 ? 1 : fraction <= 2 ? 2 : fraction <= 2.5 ? 2.5 : fraction <= 5 ? 5 : 10;
  return nice * power;
}

/**
 * Round tick values covering [min, max] with about `count` intervals. The
 * first and last ticks bound the axis, so the data never touches the edges
 * of the plot at an arbitrary value.
 */
export function niceTicks(min: number, max: number, count = 5): { ticks: number[]; step: number } {
  if (min === max) {
    const pad = Math.abs(min) * 0.1 || 1;
    return niceTicks(min - pad, max + pad, count);
  }
  const step = niceStep(max - min, count);
  const decimals = decimalsOf(step);
  const first = Math.floor(min / step);
  const last = Math.ceil(max / step);
  const ticks: number[] = [];
  for (let i = first; i <= last; i++) ticks.push(Number((i * step).toFixed(decimals)));
  return { ticks, step };
}

const MONTH_STEPS = [1, 2, 3, 6, 12, 24, 60, 120];
const DAY_MS = 86_400_000;

export interface TimeTick {
  time: number;
  label: string;
}

/**
 * Ticks on month starts between two UTC timestamps, at most `maxTicks` of
 * them. Steps of a year or more fall on January and are labelled "2026";
 * shorter steps are labelled "set/26".
 */
export function timeTicks(from: number, to: number, maxTicks: number): TimeTick[] {
  const spanMonths = (to - from) / (30.44 * DAY_MS);
  const step = MONTH_STEPS.find((s) => spanMonths / s <= maxTicks) ?? 120;

  const start = new Date(from);
  // Months since year 0 of the first month start at or after `from`.
  let index = start.getUTCFullYear() * 12 + start.getUTCMonth();
  if (from > Date.UTC(start.getUTCFullYear(), start.getUTCMonth(), 1)) index += 1;
  index = Math.ceil(index / step) * step;

  const ticks: TimeTick[] = [];
  for (;;) {
    const time = Date.UTC(Math.floor(index / 12), index % 12, 1);
    if (time > to) break;
    const iso = fromTime(time);
    ticks.push({ time, label: step >= 12 ? iso.slice(0, 4) : formatMonthShort(iso) });
    index += step;
  }
  return ticks;
}
