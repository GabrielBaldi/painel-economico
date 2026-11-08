import { useEffect, useState } from 'react';

import { parseState, serializeState, type ViewState } from '../lib/urlState';

/**
 * The view state, read from the address bar once and written back on every
 * change. replaceState keeps the back button from stepping through filters.
 */
export function useUrlState(): [ViewState, (next: ViewState) => void] {
  const [state, setState] = useState(() => parseState(window.location.search));

  useEffect(() => {
    window.history.replaceState(null, '', serializeState(state));
  }, [state]);

  return [state, setState];
}
