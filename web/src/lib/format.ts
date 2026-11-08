// Brazilian Portuguese formatting for values and dates.

import type { Frequency } from './types';

const MONTHS = ['jan', 'fev', 'mar', 'abr', 'mai', 'jun', 'jul', 'ago', 'set', 'out', 'nov', 'dez'];

const numberFormats = new Map<number, Intl.NumberFormat>();

/** Formats a number the Brazilian way (1.234,56) with fixed decimals. */
export function formatNumber(value: number, digits: number): string {
  let format = numberFormats.get(digits);
  if (!format) {
    format = new Intl.NumberFormat('pt-BR', {
      minimumFractionDigits: digits,
      maximumFractionDigits: digits,
    });
    numberFormats.set(digits, format);
  }
  // Intl uses a hyphen-minus; a real minus sign reads better next to digits.
  return format.format(value).replace('-', '−');
}

/** Exchange rates get four decimals (as the PTAX is published); rates get two. */
export function defaultDigits(unit: string): number {
  return unit === 'R$' ? 4 : 2;
}

/**
 * Formats a value with its unit: "13,75% a.a.", "0,82% no mês", "5,2%",
 * "R$ 5,0119".
 */
export function formatValue(value: number, unit: string, digits = defaultDigits(unit)): string {
  const number = formatNumber(value, digits);
  if (unit.startsWith('%')) return `${number}${unit}`;
  if (unit === 'R$') return `R$ ${number}`;
  return `${number} ${unit}`;
}

/** "07/11/2026" for daily series, "set/2026" for monthly ones. */
export function formatDate(iso: string, frequency: Frequency): string {
  const [year, month, day] = iso.split('-');
  if (frequency === 'monthly') return `${MONTHS[Number(month) - 1] ?? month}/${year}`;
  return `${day}/${month}/${year}`;
}

/** Short month label for chart axes: "set/26". */
export function formatMonthShort(iso: string): string {
  const [year = '', month] = iso.split('-');
  return `${MONTHS[Number(month) - 1] ?? month}/${year.slice(2)}`;
}

/** A signed change: "+0,50 p.p." for rates, "−3,2%" for prices. */
export function formatChange(change: number, kind: 'points' | 'percent'): string {
  const sign = change > 0 ? '+' : '';
  return kind === 'points'
    ? `${sign}${formatNumber(change, 2)} p.p.`
    : `${sign}${formatNumber(change, 1)}%`;
}
