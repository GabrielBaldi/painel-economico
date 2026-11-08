// Geometry of the line chart: scales, panels, SVG paths and the readings
// under the crosshair. Pure functions, so the component only renders.

import { toTime } from './dates';
import { formatDate, formatNumber, formatValue } from './format';
import { decimalsOf, linearScale, niceTicks, timeTicks, type TimeTick } from './scale';
import { lastAtOrBefore } from './transform';
import type { Indicator, Point } from './types';

export interface ChartSeries {
  role: 'a' | 'b';
  indicator: Indicator;
  points: Point[];
}

export interface PlottedSeries extends ChartSeries {
  times: number[];
  path: string;
  y: (value: number) => number;
}

interface Panel {
  top: number;
  bottom: number;
  label: string;
  ticks: { value: number; y: number; label: string }[];
  zeroY: number | null;
  series: PlottedSeries[];
}

export interface Layout {
  width: number;
  height: number;
  left: number;
  right: number;
  x: (time: number) => number;
  invertX: (pixel: number) => number;
  xTicks: TimeTick[];
  panels: Panel[];
  series: PlottedSeries[]; // in role order: the primary series first
}

/**
 * Builds the chart geometry. Series with the same unit share one panel and
 * one y-axis; series with different units get one panel each, stacked over
 * the same time axis (small multiples), instead of two y-axes on one plot.
 */
export function buildLayout(series: ChartSeries[], width: number): Layout {
  const compact = width < 560;
  const margin = { top: 30, right: compact ? 10 : 18, bottom: 28, left: compact ? 42 : 52 };
  const sharedAxis = series.every((s) => s.indicator.unit === series[0]?.indicator.unit);
  const groups = sharedAxis ? [series] : series.map((s) => [s]);
  const panelHeight = groups.length === 1 ? (compact ? 250 : 320) : compact ? 160 : 190;
  const gap = 44;

  const times = series.map((s) => s.points.map((p) => toTime(p.date)));
  const allTimes = times.flat();
  const t0 = Math.min(...allTimes);
  const t1 = Math.max(...allTimes);
  const left = margin.left;
  const right = width - margin.right;
  const x = linearScale([t0, t1], [left, right]);
  const invertX = linearScale([left, right], [t0, t1]);

  const plotted: PlottedSeries[] = [];
  const panels = groups.map((group, i): Panel => {
    const top = margin.top + i * (panelHeight + gap);
    const bottom = top + panelHeight;
    const values = group.flatMap((s) => s.points.map((p) => p.value));
    const { ticks, step } = niceTicks(Math.min(...values), Math.max(...values), compact ? 4 : 5);
    const lo = ticks[0] ?? 0;
    const hi = ticks.at(-1) ?? 1;
    const y = linearScale([lo, hi], [bottom, top]);

    const panelSeries = group.map((s): PlottedSeries => {
      const seriesTimes = times[series.indexOf(s)] ?? [];
      const path = s.points
        .map(
          (p, j) =>
            `${j === 0 ? 'M' : 'L'}${x(seriesTimes[j] ?? t0).toFixed(1)},${y(p.value).toFixed(1)}`,
        )
        .join('');
      const result = { ...s, times: seriesTimes, path, y };
      plotted.push(result);
      return result;
    });

    const first = group[0]?.indicator;
    return {
      top,
      bottom,
      label:
        groups.length > 1 && first ? `${first.shortName} (${first.unit})` : (first?.unit ?? ''),
      ticks: ticks.map((value) => ({
        value,
        y: y(value),
        label: formatNumber(value, decimalsOf(step)),
      })),
      zeroY: lo < 0 && hi > 0 ? y(0) : null,
      series: panelSeries,
    };
  });

  const lastBottom = panels.at(-1)?.bottom ?? margin.top;
  return {
    width,
    height: lastBottom + margin.bottom,
    left,
    right,
    x,
    invertX,
    xTicks: timeTicks(t0, t1, compact ? 4 : 8),
    panels,
    series: plotted.sort((a, b) => a.role.localeCompare(b.role)),
  };
}

export interface Reading {
  series: PlottedSeries;
  point: Point;
  time: number;
}

/** The values shown for the primary series' point at `index`. */
export function readingsAt(layout: Layout, index: number): Reading[] {
  const [primary, ...others] = layout.series;
  const time = primary?.times[index];
  const point = primary?.points[index];
  if (!primary || time === undefined || !point) return [];
  const readings: Reading[] = [{ series: primary, point, time }];
  for (const s of others) {
    // The other series shows the value in force on that date.
    const j = lastAtOrBefore(s.times, time);
    const p = s.points[j];
    const t = s.times[j];
    if (p && t !== undefined) readings.push({ series: s, point: p, time: t });
  }
  return readings;
}

export function readingText(r: Reading, primaryDate: string): string {
  const value = formatValue(r.point.value, r.series.indicator.unit);
  const own =
    r.point.date !== primaryDate
      ? ` (${formatDate(r.point.date, r.series.indicator.frequency)})`
      : '';
  return `${r.series.indicator.shortName} ${value}${own}`;
}
