import { describe, expect, it } from 'vitest';

import { buildLayout, readingText, readingsAt, type ChartSeries } from './chartLayout';
import type { Frequency, Indicator } from './types';

function series(
  role: 'a' | 'b',
  id: string,
  unit: string,
  frequency: Frequency,
  points: [string, number][],
): ChartSeries {
  const indicator: Indicator = {
    id,
    name: id,
    shortName: id,
    unit,
    frequency,
    description: '',
    source: '',
    sourceUrl: '',
  };
  return { role, indicator, points: points.map(([date, value]) => ({ date, value })) };
}

const selic = series('a', 'Selic', '% a.a.', 'daily', [
  ['2026-01-15', 15],
  ['2026-02-15', 14.75],
  ['2026-03-15', 14.5],
]);
const ipca12m = series('b', 'IPCA 12 meses', '% a.a.', 'monthly', [
  ['2026-01-01', 4.4],
  ['2026-02-01', 3.8],
]);
const dolar = series('b', 'Dólar', 'R$', 'daily', [
  ['2026-01-15', 5.3],
  ['2026-03-15', 5.5],
]);

describe('buildLayout', () => {
  it('puts series with the same unit on one shared axis', () => {
    const layout = buildLayout([selic, ipca12m], 800);
    expect(layout.panels).toHaveLength(1);
    expect(layout.panels[0]?.series.map((s) => s.role)).toEqual(['a', 'b']);
    expect(layout.panels[0]?.label).toBe('% a.a.');
  });

  it('stacks one panel per series when the units differ', () => {
    const layout = buildLayout([selic, dolar], 800);
    expect(layout.panels.map((p) => p.label)).toEqual(['Selic (% a.a.)', 'Dólar (R$)']);
    const [top, bottom] = layout.panels;
    expect(bottom?.top).toBeGreaterThan(top?.bottom ?? Infinity);
  });

  it('spans the x-axis from the first to the last date', () => {
    const layout = buildLayout([selic], 800);
    const path = layout.series[0]?.path ?? '';
    expect(path.startsWith(`M${layout.left.toFixed(1)},`)).toBe(true);
    expect(path.includes(`L${layout.right.toFixed(1)},`)).toBe(true);
  });

  it('bounds each y-axis with round ticks', () => {
    const ticks = buildLayout([selic], 800).panels[0]?.ticks.map((t) => t.label);
    expect(ticks?.[0]).toBe('14,5');
    expect(ticks?.at(-1)).toBe('15,0');
  });
});

describe('readingsAt', () => {
  it('pairs the primary point with the value of the other series in force on that date', () => {
    const layout = buildLayout([selic, ipca12m], 800);
    const readings = readingsAt(layout, 2); // 2026-03-15
    expect(readings.map((r) => [r.series.role, r.point.date, r.point.value])).toEqual([
      ['a', '2026-03-15', 14.5],
      ['b', '2026-02-01', 3.8],
    ]);
    const [primary, other] = readings;
    expect(other && readingText(other, primary?.point.date ?? '')).toBe(
      'IPCA 12 meses 3,80% a.a. (fev/2026)',
    );
  });

  it('returns nothing for an index outside the series', () => {
    expect(readingsAt(buildLayout([selic], 800), 99)).toEqual([]);
  });
});
