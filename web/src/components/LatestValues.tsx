import { useSeries } from '../hooks/useApi';
import { addMonths } from '../lib/dates';
import { formatChange, formatDate, formatNumber } from '../lib/format';
import { yearChange } from '../lib/transform';
import type { Indicator } from '../lib/types';

interface LatestValuesProps {
  indicators: Indicator[];
  selected: string;
  compared: string | null;
  today: string;
  onSelect: (id: string) => void;
}

/** One card per indicator with its latest value and the change in 12 months. */
export function LatestValues({
  indicators,
  selected,
  compared,
  today,
  onSelect,
}: LatestValuesProps) {
  return (
    <section className="latest" aria-labelledby="latest-title">
      <div className="section-head">
        <h2 id="latest-title">Últimos valores</h2>
        <p>Com a variação em 12 meses. Escolha um para ver no gráfico.</p>
      </div>
      <ul className="cards">
        {indicators.map((indicator) => (
          <li key={indicator.id}>
            <IndicatorCard
              indicator={indicator}
              role={indicator.id === selected ? 'a' : indicator.id === compared ? 'b' : null}
              today={today}
              onSelect={onSelect}
            />
          </li>
        ))}
      </ul>
    </section>
  );
}

interface IndicatorCardProps {
  indicator: Indicator;
  /** Which chart series this indicator is, if any. */
  role: 'a' | 'b' | null;
  today: string;
  onSelect: (id: string) => void;
}

function IndicatorCard({ indicator, role, today, onSelect }: IndicatorCardProps) {
  // Monthly series are published up to three months late, so 18 months of
  // history always include the value in force a year before the latest one.
  const { data, loading, error } = useSeries(indicator.id, addMonths(today, -18), today);
  const points = data?.points ?? [];
  const latest = points.at(-1);
  const change = yearChange(points, indicator.unit);
  const isMoney = indicator.unit === 'R$';

  return (
    <button
      type="button"
      className="card"
      data-role={role ?? undefined}
      aria-pressed={role === 'a'}
      onClick={() => {
        onSelect(indicator.id);
      }}
    >
      <span className="card-name">{indicator.shortName}</span>
      {latest ? (
        <>
          <span className="card-value">
            {isMoney && <small>R$ </small>}
            {formatNumber(latest.value, 2)}
            {!isMoney && <small>{indicator.unit}</small>}
          </span>
          <span className="card-meta">{formatDate(latest.date, indicator.frequency)}</span>
          <span className="card-change">
            {change && (
              <>
                <span aria-hidden="true">
                  {change.change > 0 ? '▲' : change.change < 0 ? '▼' : '■'}
                </span>{' '}
                {formatChange(change.change, change.kind)}
                <span className="sr-only"> em 12 meses</span>
              </>
            )}
          </span>
        </>
      ) : (
        <span className="card-meta">
          {loading ? 'Carregando…' : error ? 'Indisponível no momento' : 'Sem dados recentes'}
        </span>
      )}
    </button>
  );
}
