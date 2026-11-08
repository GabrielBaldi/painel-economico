// The whole view lives in the query string, so any chart can be shared as a
// link. Parameter names are Portuguese because people read and share them:
// ?indicador=selic&comparar=ipca-12m&periodo=10a&tabela=1
// ?indicador=dolar&comparar=nenhum&periodo=personalizado&de=2020-01-01&ate=2022-12-31

import { addMonths, isISODate } from './dates';

export type Preset = '12m' | '5a' | '10a';

export type Period =
  { kind: 'preset'; preset: Preset } | { kind: 'custom'; from: string; to: string };

export interface ViewState {
  indicator: string;
  /** Second indicator to compare with, or null for a single series. */
  compare: string | null;
  period: Period;
  table: boolean;
}

export const PRESETS: readonly { id: Preset; label: string; months: number }[] = [
  { id: '12m', label: '12 meses', months: 12 },
  { id: '5a', label: '5 anos', months: 60 },
  { id: '10a', label: '10 anos', months: 120 },
];

/** The view of a link without parameters: Selic against inflation. */
export const DEFAULT_STATE: ViewState = {
  indicator: 'selic',
  compare: 'ipca-12m',
  period: { kind: 'preset', preset: '10a' },
  table: false,
};

const NO_COMPARISON = 'nenhum';
const CUSTOM = 'personalizado';
const ID = /^[a-z0-9-]{1,40}$/;

/**
 * Reads a view from a query string. Invalid values fall back to defaults
 * instead of failing, since links get truncated and edited by hand. A link
 * that names an indicator but no comparison shows that indicator alone.
 */
export function parseState(search: string): ViewState {
  const params = new URLSearchParams(search);
  if (params.size === 0) return DEFAULT_STATE;

  const indicatorParam = params.get('indicador') ?? '';
  const indicator = ID.test(indicatorParam) ? indicatorParam : DEFAULT_STATE.indicator;

  const compareParam = params.get('comparar') ?? '';
  const compare =
    ID.test(compareParam) && compareParam !== NO_COMPARISON && compareParam !== indicator
      ? compareParam
      : null;

  return { indicator, compare, period: parsePeriod(params), table: params.get('tabela') === '1' };
}

function parsePeriod(params: URLSearchParams): Period {
  const value = params.get('periodo');
  if (value === CUSTOM) {
    const from = params.get('de') ?? '';
    const to = params.get('ate') ?? '';
    if (isISODate(from) && isISODate(to) && from <= to) {
      return { kind: 'custom', from, to };
    }
  }
  const preset = PRESETS.find((p) => p.id === value);
  return preset ? { kind: 'preset', preset: preset.id } : DEFAULT_STATE.period;
}

/** Writes a view as a query string, always with every parameter. */
export function serializeState(state: ViewState): string {
  const params = new URLSearchParams({
    indicador: state.indicator,
    comparar: state.compare ?? NO_COMPARISON,
  });
  if (state.period.kind === 'custom') {
    params.set('periodo', CUSTOM);
    params.set('de', state.period.from);
    params.set('ate', state.period.to);
  } else {
    params.set('periodo', state.period.preset);
  }
  if (state.table) params.set('tabela', '1');
  return `?${params.toString()}`;
}

/** Turns a period into the from/to dates (inclusive) to request. */
export function resolveRange(period: Period, today: string): { from: string; to: string } {
  if (period.kind === 'custom') return { from: period.from, to: period.to };
  const months = PRESETS.find((p) => p.id === period.preset)?.months ?? 12;
  return { from: addMonths(today, -months), to: today };
}

/**
 * Replaces indicator IDs the API does not know (an old or mistyped link)
 * once the catalog is loaded.
 */
export function withKnownIndicators(state: ViewState, knownIds: readonly string[]): ViewState {
  const fallback = knownIds.includes(DEFAULT_STATE.indicator)
    ? DEFAULT_STATE.indicator
    : (knownIds[0] ?? state.indicator);
  const indicator = knownIds.includes(state.indicator) ? state.indicator : fallback;
  const compare =
    state.compare !== null && knownIds.includes(state.compare) && state.compare !== indicator
      ? state.compare
      : null;
  if (indicator === state.indicator && compare === state.compare) return state;
  return { ...state, indicator, compare };
}
