import { useId, useState } from 'react';

import { isISODate } from '../lib/dates';
import type { Indicator } from '../lib/types';
import { PRESETS, type Preset, type ViewState } from '../lib/urlState';

/** The earliest date the API serves (see indicator.HistoryStart in Go). */
export const MIN_DATE = '2000-01-01';

interface ControlsProps {
  state: ViewState;
  indicators: Indicator[];
  range: { from: string; to: string };
  today: string;
  onChange: (next: ViewState) => void;
}

export function Controls({ state, indicators, range, today, onChange }: ControlsProps) {
  const id = useId();

  function selectIndicator(indicator: string) {
    // Picking the indicator being compared swaps the two.
    const compare = indicator === state.compare ? state.indicator : state.compare;
    onChange({ ...state, indicator, compare });
  }

  function selectPeriod(value: Preset | 'custom') {
    onChange({
      ...state,
      period: value === 'custom' ? { kind: 'custom', ...range } : { kind: 'preset', preset: value },
    });
  }

  const periodValue = state.period.kind === 'custom' ? 'custom' : state.period.preset;
  const periodOptions = [...PRESETS, { id: 'custom', label: 'Personalizado' } as const];

  return (
    <form
      className="controls"
      aria-label="Opções do gráfico"
      onSubmit={(event) => {
        event.preventDefault();
      }}
    >
      <div className="field">
        <label htmlFor={`${id}-indicator`}>Indicador</label>
        <select
          id={`${id}-indicator`}
          value={state.indicator}
          onChange={(event) => {
            selectIndicator(event.target.value);
          }}
        >
          {indicators.map((i) => (
            <option key={i.id} value={i.id}>
              {i.shortName}
            </option>
          ))}
        </select>
      </div>

      <div className="field">
        <label htmlFor={`${id}-compare`}>Comparar com</label>
        <select
          id={`${id}-compare`}
          value={state.compare ?? ''}
          onChange={(event) => {
            onChange({ ...state, compare: event.target.value || null });
          }}
        >
          <option value="">Nenhum</option>
          {indicators
            .filter((i) => i.id !== state.indicator)
            .map((i) => (
              <option key={i.id} value={i.id}>
                {i.shortName}
              </option>
            ))}
        </select>
      </div>

      <fieldset className="field period">
        <legend>Período</legend>
        <div className="segmented">
          {periodOptions.map((option) => (
            <label key={option.id} className="segment">
              <input
                type="radio"
                name={`${id}-period`}
                value={option.id}
                checked={periodValue === option.id}
                onChange={() => {
                  selectPeriod(option.id);
                }}
              />
              <span>{option.label}</span>
            </label>
          ))}
        </div>
      </fieldset>

      {state.period.kind === 'custom' && (
        <CustomRange
          from={state.period.from}
          to={state.period.to}
          today={today}
          onApply={(from, to) => {
            onChange({ ...state, period: { kind: 'custom', from, to } });
          }}
        />
      )}
    </form>
  );
}

interface CustomRangeProps {
  from: string;
  to: string;
  today: string;
  onApply: (from: string, to: string) => void;
}

/**
 * Two date inputs. Each edit is applied as soon as both dates are valid and
 * in order; until then the draft stays local and the error is announced.
 */
function CustomRange({ from, to, today, onApply }: CustomRangeProps) {
  const id = useId();
  const [draft, setDraft] = useState({ from, to });
  const complete = isISODate(draft.from) && isISODate(draft.to);
  const error = !complete
    ? 'Preencha as duas datas.'
    : draft.from > draft.to
      ? 'A data inicial deve ser anterior à final.'
      : null;

  function update(next: { from: string; to: string }) {
    setDraft(next);
    if (isISODate(next.from) && isISODate(next.to) && next.from <= next.to) {
      onApply(next.from, next.to);
    }
  }

  return (
    <div className="custom-range">
      <div className="field">
        <label htmlFor={`${id}-from`}>De</label>
        <input
          id={`${id}-from`}
          type="date"
          min={MIN_DATE}
          max={today}
          value={draft.from}
          aria-invalid={error !== null}
          aria-describedby={error ? `${id}-error` : undefined}
          onChange={(event) => {
            update({ ...draft, from: event.target.value });
          }}
        />
      </div>
      <div className="field">
        <label htmlFor={`${id}-to`}>Até</label>
        <input
          id={`${id}-to`}
          type="date"
          min={MIN_DATE}
          max={today}
          value={draft.to}
          aria-invalid={error !== null}
          aria-describedby={error ? `${id}-error` : undefined}
          onChange={(event) => {
            update({ ...draft, to: event.target.value });
          }}
        />
      </div>
      <p className="field-error" id={`${id}-error`} role="alert">
        {error}
      </p>
    </div>
  );
}
