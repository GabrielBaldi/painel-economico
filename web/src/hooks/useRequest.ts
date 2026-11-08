import { useCallback, useEffect, useState } from 'react';

export interface RequestState<T> {
  /** Latest response. While a new request runs, it still holds the previous one. */
  data: T | null;
  loading: boolean;
  error: string | null;
  retry: () => void;
}

interface Settled<T> {
  key: string;
  attempt: number;
  data: T | null;
  error: string | null;
}

const NETWORK_ERROR = 'Não foi possível falar com o servidor. Verifique sua conexão.';

/**
 * Runs `load` whenever `key` changes (null means "nothing to load") and
 * aborts the previous request. Loading is derived by comparing the key of
 * the last settled request with the current one, so no state is set
 * synchronously inside the effect.
 */
export function useRequest<T>(
  key: string | null,
  load: (signal: AbortSignal) => Promise<T>,
): RequestState<T> {
  const [attempt, setAttempt] = useState(0);
  const [settled, setSettled] = useState<Settled<T> | null>(null);

  useEffect(() => {
    if (key === null) return;
    const controller = new AbortController();
    load(controller.signal).then(
      (data) => {
        setSettled({ key, attempt, data, error: null });
      },
      (error: unknown) => {
        if (controller.signal.aborted) return;
        // fetch rejects with a TypeError when the network fails.
        const message =
          error instanceof TypeError
            ? NETWORK_ERROR
            : error instanceof Error
              ? error.message
              : String(error);
        setSettled({ key, attempt, data: null, error: message });
      },
    );
    return () => {
      controller.abort();
    };
    // `load` is recreated on every render; `key` already identifies the request.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key, attempt]);

  const retry = useCallback(() => {
    setAttempt((n) => n + 1);
  }, []);

  const current = settled?.key === key && settled.attempt === attempt;
  return {
    data: settled?.data ?? null,
    loading: key !== null && !current,
    error: current ? settled.error : null,
    retry,
  };
}
