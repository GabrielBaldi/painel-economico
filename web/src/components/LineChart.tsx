import { useId, useMemo, useRef, useState, type KeyboardEvent, type PointerEvent } from 'react';

import { useElementWidth } from '../hooks/useElementWidth';
import { buildLayout, readingText, readingsAt, type ChartSeries } from '../lib/chartLayout';
import { formatDate, formatValue } from '../lib/format';
import { nearestIndex } from '../lib/transform';

interface LineChartProps {
  series: ChartSeries[];
  /** Id of the element that describes the chart for screen readers. */
  describedBy: string;
}

export function LineChart({ series, describedBy }: LineChartProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const width = useElementWidth(containerRef);
  const [active, setActive] = useState<number | null>(null);
  const hintId = useId();

  const layout = useMemo(() => (width > 0 ? buildLayout(series, width) : null), [series, width]);
  const primary = layout?.series[0];
  const count = primary?.points.length ?? 0;
  const index = active !== null && active < count ? active : null;
  const readings = layout && index !== null ? readingsAt(layout, index) : [];
  const head = readings[0];

  function onPointerMove(event: PointerEvent<SVGRectElement>) {
    if (!layout || !primary) return;
    const box = event.currentTarget.ownerSVGElement?.getBoundingClientRect();
    if (!box) return;
    setActive(nearestIndex(primary.times, layout.invertX(event.clientX - box.left)));
  }

  function onKeyDown(event: KeyboardEvent<HTMLDivElement>) {
    if (count === 0) return;
    const current = index ?? count - 1;
    const page = Math.max(1, Math.round(count / 20));
    const next: Record<string, number> = {
      ArrowLeft: current - 1,
      ArrowDown: current - 1,
      ArrowRight: current + 1,
      ArrowUp: current + 1,
      PageDown: current - page,
      PageUp: current + page,
      Home: 0,
      End: count - 1,
    };
    const target = next[event.key];
    if (target === undefined) return;
    event.preventDefault();
    setActive(Math.min(count - 1, Math.max(0, target)));
  }

  const valueText = head
    ? `${formatDate(head.point.date, head.series.indicator.frequency)}: ${readings
        .map((r) => readingText(r, head.point.date))
        .join('; ')}`
    : 'Nenhum ponto selecionado';
  const crosshairX = head && layout ? layout.x(head.time) : null;

  return (
    <div className="chart" ref={containerRef}>
      <div
        className="chart-focus"
        role="slider"
        tabIndex={0}
        aria-label="Valores do gráfico por data"
        aria-describedby={`${describedBy} ${hintId}`}
        aria-orientation="horizontal"
        aria-valuemin={0}
        aria-valuemax={Math.max(0, count - 1)}
        aria-valuenow={index ?? Math.max(0, count - 1)}
        aria-valuetext={valueText}
        onKeyDown={onKeyDown}
        onFocus={() => {
          setActive((current) => current ?? count - 1);
        }}
        onBlur={() => {
          setActive(null);
        }}
      >
        {layout && (
          <svg
            width={layout.width}
            height={layout.height}
            viewBox={`0 0 ${layout.width} ${layout.height}`}
            aria-hidden="true"
            focusable="false"
          >
            {layout.panels.map((panel) => (
              <g key={panel.top}>
                <text className="chart-unit" x={layout.left} y={panel.top - 14}>
                  {panel.label}
                </text>
                {panel.ticks.map((tick) => (
                  <g key={tick.value}>
                    <line
                      className="chart-grid"
                      x1={layout.left}
                      x2={layout.right}
                      y1={tick.y}
                      y2={tick.y}
                    />
                    <text
                      className="chart-tick"
                      x={layout.left - 8}
                      y={tick.y}
                      dy="0.32em"
                      textAnchor="end"
                    >
                      {tick.label}
                    </text>
                  </g>
                ))}
                {panel.zeroY !== null && (
                  <line
                    className="chart-zero"
                    x1={layout.left}
                    x2={layout.right}
                    y1={panel.zeroY}
                    y2={panel.zeroY}
                  />
                )}
                <line
                  className="chart-baseline"
                  x1={layout.left}
                  x2={layout.right}
                  y1={panel.bottom}
                  y2={panel.bottom}
                />
                {layout.xTicks.map((tick) => (
                  <line
                    key={tick.time}
                    className="chart-baseline"
                    x1={layout.x(tick.time)}
                    x2={layout.x(tick.time)}
                    y1={panel.bottom}
                    y2={panel.bottom + 4}
                  />
                ))}
                {panel.series.map((s) => {
                  const last = s.points.at(-1);
                  const lastTime = s.times.at(-1);
                  return (
                    <g key={s.role}>
                      <path className={`chart-line chart-line-${s.role}`} d={s.path} />
                      {last && lastTime !== undefined && (
                        <circle
                          className={`chart-dot chart-dot-${s.role}`}
                          cx={layout.x(lastTime)}
                          cy={s.y(last.value)}
                          r={4}
                        />
                      )}
                    </g>
                  );
                })}
              </g>
            ))}

            {layout.xTicks.map((tick) => (
              <text
                key={tick.time}
                className="chart-tick"
                x={layout.x(tick.time)}
                y={layout.height - 8}
                textAnchor="middle"
              >
                {tick.label}
              </text>
            ))}

            {crosshairX !== null && (
              <g>
                <line
                  className="chart-crosshair"
                  x1={crosshairX}
                  x2={crosshairX}
                  y1={layout.panels[0]?.top ?? 0}
                  y2={layout.panels.at(-1)?.bottom ?? 0}
                />
                {readings.map((r) => (
                  <circle
                    key={r.series.role}
                    className={`chart-dot chart-dot-${r.series.role}`}
                    cx={layout.x(r.time)}
                    cy={r.series.y(r.point.value)}
                    r={5}
                  />
                ))}
              </g>
            )}

            <rect
              className="chart-hit"
              x={layout.left}
              y={0}
              width={Math.max(0, layout.right - layout.left)}
              height={layout.height}
              onPointerMove={onPointerMove}
              onPointerDown={onPointerMove}
              onPointerLeave={() => {
                setActive(null);
              }}
            />
          </svg>
        )}

        {head && layout && crosshairX !== null && (
          <div
            className={`chart-tooltip${crosshairX > layout.width / 2 ? ' chart-tooltip-left' : ''}`}
            style={{ left: crosshairX }}
            aria-hidden="true"
          >
            <div className="chart-tooltip-date">
              {formatDate(head.point.date, head.series.indicator.frequency)}
            </div>
            {readings.map((r) => (
              <div key={r.series.role} className="chart-tooltip-row">
                <svg className="line-key" width="18" height="8" aria-hidden="true">
                  <line
                    className={`chart-line chart-line-${r.series.role}`}
                    x1="1"
                    x2="17"
                    y1="4"
                    y2="4"
                  />
                </svg>
                <strong>{formatValue(r.point.value, r.series.indicator.unit)}</strong>
                <span>
                  {r.series.indicator.shortName}
                  {r.point.date !== head.point.date &&
                    ` · ${formatDate(r.point.date, r.series.indicator.frequency)}`}
                </span>
              </div>
            ))}
          </div>
        )}
      </div>
      <p className="chart-hint" id={hintId}>
        Passe o mouse ou o dedo sobre o gráfico, ou use as setas do teclado, para ler cada valor.
      </p>
    </div>
  );
}
