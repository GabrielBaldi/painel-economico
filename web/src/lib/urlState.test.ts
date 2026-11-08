import { describe, expect, it } from 'vitest';

import {
  DEFAULT_STATE,
  parseState,
  resolveRange,
  serializeState,
  withKnownIndicators,
  type ViewState,
} from './urlState';

describe('parseState', () => {
  it.each<[string, string, ViewState]>([
    ['empty link shows the default comparison', '', DEFAULT_STATE],
    [
      'full link',
      '?indicador=dolar&comparar=selic&periodo=5a&tabela=1',
      {
        indicator: 'dolar',
        compare: 'selic',
        period: { kind: 'preset', preset: '5a' },
        table: true,
      },
    ],
    [
      'an indicator without comparison shows it alone',
      '?indicador=desemprego',
      {
        indicator: 'desemprego',
        compare: null,
        period: { kind: 'preset', preset: '10a' },
        table: false,
      },
    ],
    [
      'explicit "nenhum" means no comparison',
      '?indicador=selic&comparar=nenhum&periodo=12m',
      {
        indicator: 'selic',
        compare: null,
        period: { kind: 'preset', preset: '12m' },
        table: false,
      },
    ],
    [
      'comparing an indicator with itself is dropped',
      '?indicador=ipca&comparar=ipca',
      { indicator: 'ipca', compare: null, period: { kind: 'preset', preset: '10a' }, table: false },
    ],
    [
      'custom range',
      '?indicador=ipca&periodo=personalizado&de=2020-01-01&ate=2022-12-31',
      {
        indicator: 'ipca',
        compare: null,
        period: { kind: 'custom', from: '2020-01-01', to: '2022-12-31' },
        table: false,
      },
    ],
    [
      'custom range with an impossible date falls back to the default period',
      '?indicador=ipca&periodo=personalizado&de=2021-02-30&ate=2022-12-31',
      { indicator: 'ipca', compare: null, period: DEFAULT_STATE.period, table: false },
    ],
    [
      'inverted custom range falls back to the default period',
      '?indicador=ipca&periodo=personalizado&de=2023-01-01&ate=2022-12-31',
      { indicator: 'ipca', compare: null, period: DEFAULT_STATE.period, table: false },
    ],
    [
      'unknown period and malformed IDs fall back to defaults',
      '?indicador=<script>&comparar=SELIC&periodo=99a&tabela=sim',
      { indicator: 'selic', compare: null, period: DEFAULT_STATE.period, table: false },
    ],
  ])('%s', (_, search, expected) => {
    expect(parseState(search)).toEqual(expected);
  });
});

describe('serializeState', () => {
  it('writes every parameter so links do not depend on defaults', () => {
    expect(serializeState(DEFAULT_STATE)).toBe('?indicador=selic&comparar=ipca-12m&periodo=10a');
  });

  it('writes custom ranges, "nenhum" and the table flag', () => {
    const state: ViewState = {
      indicator: 'dolar',
      compare: null,
      period: { kind: 'custom', from: '2020-03-01', to: '2020-12-31' },
      table: true,
    };
    expect(serializeState(state)).toBe(
      '?indicador=dolar&comparar=nenhum&periodo=personalizado&de=2020-03-01&ate=2020-12-31&tabela=1',
    );
  });

  it.each<ViewState>([
    DEFAULT_STATE,
    { indicator: 'ipca', compare: null, period: { kind: 'preset', preset: '12m' }, table: true },
    {
      indicator: 'desemprego',
      compare: 'selic',
      period: { kind: 'custom', from: '2012-03-01', to: '2026-09-30' },
      table: false,
    },
  ])('round-trips %o', (state) => {
    expect(parseState(serializeState(state))).toEqual(state);
  });
});

describe('resolveRange', () => {
  it.each([
    ['12m', '2026-11-08', '2025-11-08'],
    ['5a', '2026-11-08', '2021-11-08'],
    ['10a', '2026-11-08', '2016-11-08'],
    ['12m', '2024-02-29', '2023-02-28'],
  ] as const)('preset %s ending %s starts %s', (preset, today, from) => {
    expect(resolveRange({ kind: 'preset', preset }, today)).toEqual({ from, to: today });
  });

  it('keeps custom ranges as they are', () => {
    const period = { kind: 'custom', from: '2020-01-01', to: '2021-06-30' } as const;
    expect(resolveRange(period, '2026-11-08')).toEqual({ from: '2020-01-01', to: '2021-06-30' });
  });
});

describe('withKnownIndicators', () => {
  const known = ['selic', 'ipca', 'dolar'];

  it('returns the same object when every ID is known', () => {
    const state = { ...DEFAULT_STATE, compare: 'ipca' };
    expect(withKnownIndicators(state, known)).toBe(state);
  });

  it('replaces an unknown indicator with the default and drops an unknown comparison', () => {
    const state = { ...DEFAULT_STATE, indicator: 'bitcoin', compare: 'ouro' };
    expect(withKnownIndicators(state, known)).toEqual({
      ...DEFAULT_STATE,
      indicator: 'selic',
      compare: null,
    });
  });

  it('drops a comparison that became equal to the replaced indicator', () => {
    const state = { ...DEFAULT_STATE, indicator: 'bitcoin', compare: 'selic' };
    expect(withKnownIndicators(state, known).compare).toBeNull();
  });
});
