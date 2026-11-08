// Shapes of the Go API responses (see internal/indicator).

export type Frequency = 'daily' | 'monthly';

export interface Indicator {
  id: string;
  name: string;
  shortName: string;
  unit: string;
  frequency: Frequency;
  description: string;
  source: string;
  sourceUrl: string;
}

/** One observation; date is ISO (YYYY-MM-DD), monthly series use day 01. */
export interface Point {
  date: string;
  value: number;
}

export interface SeriesResponse {
  indicator: Indicator;
  points: Point[];
  stale: boolean;
  fetchedAt: string;
}

export interface CatalogResponse {
  indicators: Indicator[];
}
