import { describe, expect, it } from 'vitest';

import { toTime } from './dates';
import { decimalsOf, linearScale, niceTicks, timeTicks } from './scale';

describe('linearScale', () => {
  it('maps the domain onto the range, including inverted ranges', () => {
    const y = linearScale([0, 10], [200, 0]);
    expect(y(0)).toBe(200);
    expect(y(5)).toBe(100);
    expect(y(10)).toBe(0);
  });

  it('does not divide by zero on an empty domain', () => {
    expect(linearScale([3, 3], [0, 100])(3)).toBe(0);
  });
});

describe('niceTicks', () => {
  it.each([
    { min: 2, max: 14.25, ticks: [0, 2.5, 5, 7.5, 10, 12.5, 15], step: 2.5 },
    { min: 4.85, max: 6.2, ticks: [4.5, 5, 5.5, 6, 6.5], step: 0.5 },
    { min: -0.68, max: 1.62, ticks: [-1, -0.5, 0, 0.5, 1, 1.5, 2], step: 0.5 },
    { min: 13.75, max: 13.75, ticks: [12, 13, 14, 15, 16], step: 1 },
  ])('$min..$max → step $step', ({ min, max, ticks, step }) => {
    const result = niceTicks(min, max, 5);
    expect(result.step).toBe(step);
    expect(result.ticks).toEqual(ticks);
  });

  it('always covers the data', () => {
    const { ticks } = niceTicks(0.123, 98.7, 4);
    expect(ticks[0]).toBeLessThanOrEqual(0.123);
    expect(ticks.at(-1)).toBeGreaterThanOrEqual(98.7);
  });
});

describe('decimalsOf', () => {
  it.each([
    [5, 0],
    [2.5, 1],
    [0.25, 2],
    [0.1 + 0.2, 1],
  ])('%f needs %i decimals', (step, decimals) => {
    expect(decimalsOf(step)).toBe(decimals);
  });
});

describe('timeTicks', () => {
  it('uses yearly ticks on January for a ten-year range', () => {
    const ticks = timeTicks(toTime('2016-11-08'), toTime('2026-11-08'), 6);
    expect(ticks.map((t) => t.label)).toEqual(['2018', '2020', '2022', '2024', '2026']);
  });

  it('uses month labels for a one-year range', () => {
    const ticks = timeTicks(toTime('2025-11-08'), toTime('2026-11-08'), 6);
    expect(ticks.map((t) => t.label)).toEqual([
      'jan/26',
      'mar/26',
      'mai/26',
      'jul/26',
      'set/26',
      'nov/26',
    ]);
  });

  it('includes a tick that falls exactly on the start', () => {
    const ticks = timeTicks(toTime('2026-01-01'), toTime('2026-03-15'), 4);
    expect(ticks.map((t) => t.label)).toEqual(['jan/26', 'fev/26', 'mar/26']);
  });
});
