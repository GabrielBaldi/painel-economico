// Plain-language summaries of a series, for screen readers.

import { formatDate, formatValue } from './format';
import { extremes } from './transform';
import type { Indicator, Point } from './types';

/**
 * "Selic: 14,00% a.a. em 08/11/2016 e 13,75% a.a. em 07/11/2026; mínima de
 * 2,00% a.a. em 06/08/2020 e máxima de 15,00% a.a. em 19/06/2025."
 */
export function describeSeries(indicator: Indicator, points: readonly Point[]): string {
  const e = extremes(points);
  if (!e) return `${indicator.shortName}: sem dados no período.`;
  const at = (p: Point) =>
    `${formatValue(p.value, indicator.unit)} em ${formatDate(p.date, indicator.frequency)}`;
  return (
    `${indicator.shortName}: ${at(e.first)} e ${at(e.last)}; ` +
    `mínima de ${at(e.min)} e máxima de ${at(e.max)}.`
  );
}
