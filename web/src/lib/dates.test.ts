import { describe, expect, it } from 'vitest';

import { addMonths, isISODate } from './dates';

describe('ISO date helpers', () => {
  it.each([
    ['2026-11-08', true],
    ['2024-02-29', true],
    ['2023-02-29', false],
    ['2026-13-01', false],
    ['08/11/2026', false],
    ['', false],
  ])('isISODate(%s) is %s', (value, expected) => {
    expect(isISODate(value)).toBe(expected);
  });

  it.each([
    ['2026-11-08', -12, '2025-11-08'],
    ['2026-03-31', -1, '2026-02-28'],
    ['2024-02-29', -12, '2023-02-28'],
    ['2026-11-08', -120, '2016-11-08'],
  ])('addMonths(%s, %i) is %s', (iso, months, expected) => {
    expect(addMonths(iso, months)).toBe(expected);
  });
});
