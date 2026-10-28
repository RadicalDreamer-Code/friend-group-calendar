import { focusManager } from '@tanstack/react-query';
import { useFocusEffect, useIsFocused } from 'expo-router';
import { useCallback, useEffect, useRef, useState } from 'react';
import { AppState, Platform } from 'react-native';

import { today } from './dates';

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
  useEffect(() => {
    latest.current = queries;
  });
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

/** How often useToday looks again at the latest, in case the clock or time zone changed. */
const DAY_CHECK_MAX_MS = 60 * 60 * 1000;

/**
 * The device's today as YYYY-MM-DD, re-rendering when the day changes: at
 * midnight, and when the app comes back after a night in the background
 * (timers do not run there).
 */
export function useToday(): string {
  const [day, setDay] = useState(today);
  useEffect(() => {
    let timer: ReturnType<typeof setTimeout>;
    // Looks again just after the next midnight, or within the hour: a clock
    // or time zone change moves midnight, so a check can still find the same
    // day, and the next one has to be scheduled either way.
    const schedule = () => {
      const now = new Date();
      const midnight = new Date(now.getFullYear(), now.getMonth(), now.getDate() + 1);
      // A second late, so today() is surely the new day.
      const delay = Math.min(midnight.getTime() - now.getTime() + 1000, DAY_CHECK_MAX_MS);
      timer = setTimeout(() => {
        setDay(today());
        schedule();
      }, delay);
    };
    schedule();
    const subscription = AppState.addEventListener('change', (state) => {
      if (state === 'active') setDay(today());
    });
    return () => {
      clearTimeout(timer);
      subscription.remove();
    };
  }, []);
  return day;
}
