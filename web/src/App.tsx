import { ChartPanel } from './components/ChartPanel';
import { Controls } from './components/Controls';
import { LatestValues } from './components/LatestValues';
import { useCatalog, useSeries } from './hooks/useApi';
import { useUrlState } from './hooks/useUrlState';
import { localISODate } from './lib/dates';
import { resolveRange, withKnownIndicators } from './lib/urlState';

const editionFormat = new Intl.DateTimeFormat('pt-BR', { dateStyle: 'full' });

export function App() {
  const [urlState, setState] = useUrlState();
  const catalog = useCatalog();
  const indicators = catalog.data?.indicators ?? [];
  const ready = indicators.length > 0;
  const state = ready
    ? withKnownIndicators(
        urlState,
        indicators.map((i) => i.id),
      )
    : urlState;

  const now = new Date();
  const today = localISODate(now);
  const range = resolveRange(state.period, today);
  const a = useSeries(ready ? state.indicator : null, range.from, range.to);
  const b = useSeries(ready ? state.compare : null, range.from, range.to);

  function selectFromCard(id: string) {
    const compare = id === state.compare ? state.indicator : state.compare;
    setState({ ...state, indicator: id, compare });
  }

  return (
    <>
      <a className="skip-link" href="#explorar">
        Pular para o gráfico
      </a>
      <div className="page">
        <header className="masthead">
          <p className="kicker">
            <span>Dados abertos · Banco Central e IBGE</span>
            <span className="edition">{editionFormat.format(now)}</span>
          </p>
          <h1>Painel Econômico</h1>
          <p className="lede">
            Juros, inflação, dólar e desemprego do Brasil num só lugar. Compare dois indicadores,
            escolha o período e compartilhe o gráfico com um link.
          </p>
        </header>

        <main className="layout">
          {catalog.error ? (
            <div className="notice notice-error" role="alert">
              <p>{catalog.error}</p>
              <button type="button" className="button" onClick={catalog.retry}>
                Tentar novamente
              </button>
            </div>
          ) : (
            <>
              <LatestValues
                indicators={indicators}
                selected={state.indicator}
                compared={state.compare}
                today={today}
                onSelect={selectFromCard}
              />

              <div className="explorer" id="explorar">
                <Controls
                  state={state}
                  indicators={indicators}
                  range={range}
                  today={today}
                  onChange={setState}
                />
                <ChartPanel
                  a={a.data}
                  b={state.compare ? b.data : null}
                  range={range}
                  loading={a.loading || b.loading}
                  error={a.error ?? b.error}
                  showTable={state.table}
                  onToggleTable={() => {
                    setState({ ...state, table: !state.table });
                  }}
                  onRetry={() => {
                    a.retry();
                    b.retry();
                  }}
                />
              </div>
            </>
          )}
        </main>

        <footer className="footer">
          <p>
            Dados do{' '}
            <a href="https://dadosabertos.bcb.gov.br" target="_blank" rel="noreferrer">
              Banco Central do Brasil
            </a>{' '}
            (SGS) e do{' '}
            <a href="https://sidra.ibge.gov.br" target="_blank" rel="noreferrer">
              IBGE
            </a>{' '}
            (SIDRA), com cache no servidor. As fontes podem revisar valores passados.
          </p>
          <p>Projeto de portfólio feito com Go, React e TypeScript.</p>
        </footer>
      </div>
    </>
  );
}
