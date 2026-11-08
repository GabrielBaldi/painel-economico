import { describe, expect, it } from 'vitest';

import { formatChange, formatDate, formatValue } from './format';

describe('formatValue', () => {
  it.each([
    [13.75, '% a.a.', '13,75% a.a.'],
    [-0.32, '% no mês', '−0,32% no mês'],
    [5.2, '%', '5,20%'],
    [5.0119, 'R$', 'R$ 5,0119'],
    [1234.5, 'pontos', '1.234,50 pontos'],
  ])('%f %s → %s', (value, unit, expected) => {
    expect(formatValue(value, unit)).toBe(expected);
  });

  it('accepts a custom number of decimals', () => {
    expect(formatValue(5.0119, 'R$', 2)).toBe('R$ 5,01');
  });
});

describe('formatDate', () => {
  it('shows the day for daily series and the month for monthly ones', () => {
    expect(formatDate('2026-11-07', 'daily')).toBe('07/11/2026');
    expect(formatDate('2026-09-01', 'monthly')).toBe('set/2026');
  });
});

describe('formatChange', () => {
  it.each([
    [0.5, 'points', '+0,50 p.p.'],
    [-1.25, 'points', '−1,25 p.p.'],
    [-3.24, 'percent', '−3,2%'],
    [0, 'percent', '0,0%'],
  ] as const)('%f %s → %s', (change, kind, expected) => {
    expect(formatChange(change, kind)).toBe(expected);
  });
});
