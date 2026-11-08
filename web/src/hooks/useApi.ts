import { fetchCatalog, fetchSeries } from '../lib/api';
import type { CatalogResponse, SeriesResponse } from '../lib/types';
import { useRequest, type RequestState } from './useRequest';

export function useCatalog(): RequestState<CatalogResponse> {
  return useRequest('catalog', fetchCatalog);
}

/** Loads an indicator between two ISO dates; pass a null id to load nothing. */
export function useSeries(
  id: string | null,
  from: string,
  to: string,
): RequestState<SeriesResponse> {
  const key = id === null ? null : `${id}|${from}|${to}`;
  return useRequest(key, (signal) => fetchSeries(id ?? '', from, to, signal));
}
