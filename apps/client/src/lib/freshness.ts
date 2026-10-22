import { focusManager } from '@tanstack/react-query';
import { useFocusEffect, useIsFocused } from 'expo-router';
import { useCallback, useRef } from 'react';
import { AppState, Platform } from 'react-native';

/**
 * Keeps shared data fresh without live updates: queries refetch when the app
 * or browser tab comes back into focus, when a screen is shown again, and
 * every `pollInterval` while a calendar screen is on screen.
 */
export const pollInterval = 30_000;

/**
 * On web TanStack Query already listens for the tab becoming visible; on
 * iOS and Android the app coming to the foreground counts as focus.
 */
export function listenForAppFocus() {
  if (Platform.OS === 'web') return;
  focusManager.setEventListener((setFocused) => {
    const subscription = AppState.addEventListener('change', (state) => setFocused(state === 'active'));
    return () => subscription.remove();
  });
}

/** `refetchInterval` for queries that should poll while their screen is focused. */
export function usePollWhileFocused(): number | false {
  return useIsFocused() ? pollInterval : false;
}

type Refreshable = { isStale: boolean; refetch: () => unknown };

/**
 * Refetches stale queries when the screen is shown again. Screens stay
 * mounted under the ones pushed on top, so mounting alone does not do it.
 */
export function useRefreshOnFocus(...queries: Refreshable[]) {
  const first = useRef(true);
  const latest = useRef(queries);
  latest.current = queries;
  useFocusEffect(
    useCallback(() => {
      if (first.current) {
        first.current = false;
        return;
      }
      for (const q of latest.current) if (q.isStale) q.refetch();
    }, []),
  );
}
