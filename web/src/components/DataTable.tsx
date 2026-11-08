import { useId, useMemo } from 'react';

import { defaultDigits, formatDate, formatNumber } from '../lib/format';
import { toTableRows, type Series } from '../lib/transform';

interface DataTableProps {
  a: Series;
  b: Series | null;
}

/** The chart's data as an accessible table, newest rows first. */
export function DataTable({ a, b }: DataTableProps) {
  const captionId = useId();
  const rows = useMemo(() => toTableRows(a, b), [a, b]);
  const daily = a.indicator.frequency === 'daily' && (!b || b.indicator.frequency === 'daily');
  const mixed = b !== null && a.indicator.frequency !== b.indicator.frequency;
  // Units are in the column headers, so cells hold just the numbers.
  const cell = (value: number | null, s: Series) =>
    value === null ? '—' : formatNumber(value, defaultDigits(s.indicator.unit));

  return (
    // The region is focusable so keyboard users can scroll it.
    // eslint-disable-next-line jsx-a11y/no-noninteractive-tabindex
    <div className="table-scroll" role="region" aria-labelledby={captionId} tabIndex={0}>
      <table className="data-table">
        <caption id={captionId}>
          {b ? `${a.indicator.shortName} e ${b.indicator.shortName}` : a.indicator.name}, do mais
          recente para o mais antigo ({rows.length} linhas).
          {mixed && ' Os valores diários aparecem como o último valor de cada mês.'}
        </caption>
        <thead>
          <tr>
            <th scope="col">{daily ? 'Data' : 'Mês'}</th>
            <th scope="col">
              {a.indicator.shortName} <span className="unit">({a.indicator.unit})</span>
            </th>
            {b && (
              <th scope="col">
                {b.indicator.shortName} <span className="unit">({b.indicator.unit})</span>
              </th>
            )}
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => (
            <tr key={row.date}>
              <th scope="row">{formatDate(row.date, daily ? 'daily' : 'monthly')}</th>
              <td>{cell(row.a, a)}</td>
              {b && <td>{cell(row.b, b)}</td>}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
