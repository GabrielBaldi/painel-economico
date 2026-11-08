import { describe, expect, it } from 'vitest';

import { describeSeries } from './describe';
import type { Indicator } from './types';

const selic: Indicator = {
  id: 'selic',
  name: 'Taxa Selic (meta)',
  shortName: 'Selic',
  unit: '% a.a.',
  frequency: 'daily',
  description: '',
  source: '',
  sourceUrl: '',
};

describe('describeSeries', () => {
  it('summarizes start, end and extremes', () => {
    const points = [
      { date: '2016-11-08', value: 14 },
      { date: '2020-08-06', value: 2 },
      { date: '2025-06-19', value: 15 },
      { date: '2026-11-07', value: 13.75 },
    ];
    expect(describeSeries(selic, points)).toBe(
      'Selic: 14,00% a.a. em 08/11/2016 e 13,75% a.a. em 07/11/2026; ' +
        'mínima de 2,00% a.a. em 06/08/2020 e máxima de 15,00% a.a. em 19/06/2025.',
    );
  });

  it('says when there is no data', () => {
    expect(describeSeries(selic, [])).toBe('Selic: sem dados no período.');
  });
});
