// Thin fetch wrappers for the Go API.

import type { CatalogResponse, SeriesResponse } from './types';

async function getJSON<T>(url: string, signal: AbortSignal): Promise<T> {
  const response = await fetch(url, { signal, headers: { Accept: 'application/json' } });
  if (!response.ok) {
    const body = (await response.json().catch(() => null)) as { error?: string } | null;
    throw new Error(body?.error ?? `O servidor respondeu com o código ${response.status}.`);
  }
  return (await response.json()) as T;
}

export function fetchCatalog(signal: AbortSignal): Promise<CatalogResponse> {
  return getJSON('/api/indicators', signal);
}

export function fetchSeries(
  id: string,
  from: string,
  to: string,
  signal: AbortSignal,
): Promise<SeriesResponse> {
  const query = new URLSearchParams({ from, to });
  return getJSON(`/api/indicators/${encodeURIComponent(id)}?${query.toString()}`, signal);
}
