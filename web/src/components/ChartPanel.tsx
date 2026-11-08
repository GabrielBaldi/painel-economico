import { useEffect, useId, useMemo, useState } from 'react';

import type { ChartSeries } from '../lib/chartLayout';
import { describeSeries } from '../lib/describe';
import { formatDate } from '../lib/format';
import type { SeriesResponse } from '../lib/types';
import { DataTable } from './DataTable';
import { LineChart } from './LineChart';

interface ChartPanelProps {
  a: SeriesResponse | null;
  b: SeriesResponse | null;
  range: { from: string; to: string };
  loading: boolean;
  error: string | null;
  showTable: boolean;
  onToggleTable: () => void;
  onRetry: () => void;
}

const fetchedAtFormat = new Intl.DateTimeFormat('pt-BR', {
  dateStyle: 'short',
  timeStyle: 'short',
});

export function ChartPanel(props: ChartPanelProps) {
  const { a, b, range, loading, error, showTable, onToggleTable, onRetry } = props;
  const summaryId = useId();

  // Memoized so the chart only recomputes its geometry when the data changes.
  // Nothing is drawn until the main indicator has arrived.
  const series = useMemo(() => {
    const list: ChartSeries[] = [];
    if (!a) return list;
    if (a.points.length > 0) list.push({ role: 'a', indicator: a.indicator, points: a.points });
    if (b && b.points.length > 0)
      list.push({ role: 'b', indicator: b.indicator, points: b.points });
    return list;
  }, [a, b]);
  const [first, second] = series;
  const responses = a ? [a, ...(b ? [b] : [])] : [];
  const withoutData = responses.filter((r) => r.points.length === 0);

  const title = !a
    ? 'Carregando…'
    : b
      ? `${a.indicator.shortName} × ${b.indicator.shortName}`
      : a.indicator.name;
  const period = `${formatDate(range.from, 'daily')} a ${formatDate(range.to, 'daily')}`;
  const axisNote = !second
    ? null
    : first?.indicator.unit === second.indicator.unit
      ? `Mesmo eixo, os dois em ${second.indicator.unit}`
      : 'Unidades diferentes: um painel para cada um, com o mesmo eixo de tempo';
  const staleAt = responses.find((r) => r.stale)?.fetchedAt;

  return (
    <section className="panel" aria-labelledby="chart-title" aria-busy={loading}>
      <div className="panel-head">
        <div>
          <h2 id="chart-title">{title}</h2>
          <p className="panel-sub">
            {period}
            {axisNote && <> · {axisNote}</>}
          </p>
        </div>
        <div className="panel-actions">
          <button type="button" className="button" aria-pressed={showTable} onClick={onToggleTable}>
            <span className="check" aria-hidden="true" />
            Ver como tabela
          </button>
          <CopyLinkButton />
        </div>
      </div>

      {staleAt && (
        <p className="notice notice-warn" role="status">
          A fonte não respondeu agora. Mostrando os últimos dados obtidos em{' '}
          {fetchedAtFormat.format(new Date(staleAt))}.
        </p>
      )}

      {first && withoutData.length > 0 && (
        <p className="notice notice-warn">
          Sem dados de {withoutData.map((r) => r.indicator.shortName).join(' e ')} neste período.
        </p>
      )}

      {error ? (
        <div className="notice notice-error" role="alert">
          <p>{error}</p>
          <button type="button" className="button" onClick={onRetry}>
            Tentar novamente
          </button>
        </div>
      ) : !first ? (
        <p className="placeholder">
          {loading ? 'Buscando dados no Banco Central e no IBGE…' : 'Sem dados para este período.'}
        </p>
      ) : (
        <figure className={`figure${loading ? ' is-loading' : ''}`}>
          {second && !showTable && (
            <ul className="legend" aria-label="Legenda">
              {series.map((s) => (
                <li key={s.role}>
                  <svg className="line-key" width="24" height="8" aria-hidden="true">
                    <line
                      className={`chart-line chart-line-${s.role}`}
                      x1="1"
                      x2="23"
                      y1="4"
                      y2="4"
                    />
                  </svg>
                  {s.indicator.name} <span className="unit">({s.indicator.unit})</span>
                </li>
              ))}
            </ul>
          )}
          {showTable ? (
            <DataTable a={first} b={second ?? null} />
          ) : (
            <LineChart series={series} describedBy={summaryId} />
          )}
          <figcaption className="sr-only" id={summaryId}>
            {series.map((s) => describeSeries(s.indicator, s.points)).join(' ')}
          </figcaption>
        </figure>
      )}

      {responses.length > 0 && (
        <ul className="sources" aria-label="Fontes">
          {responses.map(({ indicator }) => (
            <li key={indicator.id}>
              <strong>{indicator.shortName}:</strong> {indicator.description} Fonte:{' '}
              <a href={indicator.sourceUrl} target="_blank" rel="noreferrer">
                {indicator.source}
              </a>
              .
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

/** Copies the current address, which holds the whole view, to the clipboard. */
function CopyLinkButton() {
  const [status, setStatus] = useState<'idle' | 'copied' | 'failed'>('idle');

  useEffect(() => {
    if (status === 'idle') return;
    const timer = window.setTimeout(() => {
      setStatus('idle');
    }, 2500);
    return () => {
      window.clearTimeout(timer);
    };
  }, [status]);

  async function copy() {
    try {
      await navigator.clipboard.writeText(window.location.href);
      setStatus('copied');
    } catch {
      setStatus('failed');
    }
  }

  return (
    <>
      <button type="button" className="button button-primary" onClick={() => void copy()}>
        Copiar link
      </button>
      <span className="copy-status" role="status">
        {status === 'copied'
          ? 'Link copiado!'
          : status === 'failed'
            ? 'Não deu para copiar. Use o endereço da barra do navegador.'
            : ''}
      </span>
    </>
  );
}
