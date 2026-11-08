import { describe, expect, it } from 'vitest';

import {
  extremes,
  lastAtOrBefore,
  monthlyLast,
  nearestIndex,
  toTableRows,
  yearChange,
  type Series,
} from './transform';
import type { Frequency, Indicator, Point } from './types';

function indicator(id: string, frequency: Frequency, unit = '%'): Indicator {
  return {
    id,
    name: id,
    shortName: id,
    unit,
    frequency,
    description: '',
    source: '',
    sourceUrl: '',
  };
}

const pts = (...pairs: [string, number][]): Point[] =>
  pairs.map(([date, value]) => ({ date, value }));

describe('monthlyLast', () => {
  it('keeps the last value of each month, dated on the 1st', () => {
    const daily = pts(['2026-01-02', 1], ['2026-01-30', 2], ['2026-02-02', 3], ['2026-02-27', 4]);
    expect(monthlyLast(daily)).toEqual(pts(['2026-01-01', 2], ['2026-02-01', 4]));
  });

  it('returns nothing for nothing', () => {
    expect(monthlyLast([])).toEqual([]);
  });
});

describe('toTableRows', () => {
  const selic: Series = {
    indicator: indicator('selic', 'daily'),
    points: pts(['2026-01-15', 15], ['2026-01-31', 15], ['2026-02-27', 14.75]),
  };
  const ipca: Series = {
    indicator: indicator('ipca', 'monthly'),
    points: pts(['2026-01-01', 0.33], ['2026-02-01', 0.7], ['2026-03-01', 0.88]),
  };
  const dolar: Series = {
    indicator: indicator('dolar', 'daily', 'R$'),
    points: pts(['2026-01-30', 5.3], ['2026-02-02', 5.4]),
  };

  it('lists a single series newest first', () => {
    expect(toTableRows(ipca, null)).toEqual([
      { date: '2026-03-01', a: 0.88, b: null },
      { date: '2026-02-01', a: 0.7, b: null },
      { date: '2026-01-01', a: 0.33, b: null },
    ]);
  });

  it('reduces the daily series to month ends when frequencies differ', () => {
    expect(toTableRows(selic, ipca)).toEqual([
      { date: '2026-03-01', a: null, b: 0.88 },
      { date: '2026-02-01', a: 14.75, b: 0.7 },
      { date: '2026-01-01', a: 15, b: 0.33 },
    ]);
  });

  it('joins two daily series by date, leaving gaps where one has no value', () => {
    expect(toTableRows(selic, dolar)).toEqual([
      { date: '2026-02-27', a: 14.75, b: null },
      { date: '2026-02-02', a: null, b: 5.4 },
      { date: '2026-01-31', a: 15, b: null },
      { date: '2026-01-30', a: null, b: 5.3 },
      { date: '2026-01-15', a: 15, b: null },
    ]);
  });
});

describe('lastAtOrBefore and nearestIndex', () => {
  const xs = [10, 20, 30];

  it.each([
    [5, -1, 0],
    [10, 0, 0],
    [14, 0, 0],
    [16, 0, 1],
    [15, 0, 0], // a tie goes to the earlier point
    [30, 2, 2],
    [99, 2, 2],
  ])('target %i: as-of index %i, nearest index %i', (target, asOf, nearest) => {
    expect(lastAtOrBefore(xs, target)).toBe(asOf);
    expect(nearestIndex(xs, target)).toBe(nearest);
  });

  it('handles empty arrays', () => {
    expect(lastAtOrBefore([], 1)).toBe(-1);
    expect(nearestIndex([], 1)).toBe(-1);
  });
});

describe('yearChange', () => {
  it('measures rates in percentage points against the value in force a year earlier', () => {
    const selic = pts(['2025-11-03', 15], ['2025-11-10', 15.25], ['2026-11-07', 13.75]);
    expect(yearChange(selic, '% a.a.')).toEqual({
      latest: { date: '2026-11-07', value: 13.75 },
      base: { date: '2025-11-03', value: 15 },
      change: -1.25,
      kind: 'points',
    });
  });

  it('measures prices in percent', () => {
    const dolar = pts(['2025-09-01', 5], ['2026-09-01', 5.5]);
    const result = yearChange(dolar, 'R$');
    expect(result?.kind).toBe('percent');
    expect(result?.change).toBeCloseTo(10);
  });

  it('returns null when the series is shorter than a year', () => {
    expect(yearChange(pts(['2026-01-01', 1], ['2026-09-01', 2]), '%')).toBeNull();
    expect(yearChange([], '%')).toBeNull();
  });
});

describe('extremes', () => {
  it('finds first, last, min and max, earliest first on ties', () => {
    const points = pts(['2020-01-01', 4], ['2021-01-01', 2], ['2022-01-01', 9], ['2023-01-01', 2]);
    expect(extremes(points)).toEqual({
      first: { date: '2020-01-01', value: 4 },
      last: { date: '2023-01-01', value: 2 },
      min: { date: '2021-01-01', value: 2 },
      max: { date: '2022-01-01', value: 9 },
    });
  });

  it('returns null for an empty series', () => {
    expect(extremes([])).toBeNull();
  });
});
