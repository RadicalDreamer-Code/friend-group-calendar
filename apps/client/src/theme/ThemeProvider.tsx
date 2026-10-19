import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useLayoutEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react';
import { Platform, StyleSheet, useColorScheme } from 'react-native';

import { storage } from '@/lib/storage';

import { applyDocumentTheme, withViewTransition } from './platform';
import { palette, type ColorScheme, type Colors } from './tokens';

export type ThemePreference = 'system' | 'light' | 'dark';

const STORAGE_KEY = 'theme-preference';

type ThemeContextValue = {
  /** What the user picked. */
  preference: ThemePreference;
  /** What is actually shown (system resolved to light or dark). */
  scheme: ColorScheme;
  colors: Colors;
  setPreference: (preference: ThemePreference) => void;
};

const ThemeContext = createContext<ThemeContextValue | null>(null);

function parsePreference(value: string | null): ThemePreference | null {
  return value === 'system' || value === 'light' || value === 'dark' ? value : null;
}

export function ThemeProvider({ children }: { children: ReactNode }) {
  const systemScheme = useColorScheme();
  // Web reads the stored preference synchronously; native loads it below.
  const [preference, setPreferenceState] = useState<ThemePreference | null>(() =>
    Platform.OS === 'web' ? (parsePreference(storage.getSync(STORAGE_KEY)) ?? 'system') : null,
  );

  useEffect(() => {
    if (preference !== null) return;
    storage.get(STORAGE_KEY).then((value) => setPreferenceState(parsePreference(value) ?? 'system'));
  }, [preference]);

  const scheme: ColorScheme =
    preference === 'light' || preference === 'dark' ? preference : systemScheme === 'dark' ? 'dark' : 'light';
  const colors = palette[scheme];

  // Layout effect so the document is repainted inside the same view transition.
  useLayoutEffect(() => {
    applyDocumentTheme(scheme, colors);
  }, [scheme, colors]);

  const setPreference = useCallback((next: ThemePreference) => {
    storage.set(STORAGE_KEY, next);
    withViewTransition(() => setPreferenceState(next));
  }, []);

  const value = useMemo<ThemeContextValue | null>(
    () => (preference === null ? null : { preference, scheme, colors, setPreference }),
    [preference, scheme, colors, setPreference],
  );

  // Native only: wait the few milliseconds it takes to read the stored preference.
  if (value === null) return null;

  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
}

export function useTheme(): ThemeContextValue {
  const ctx = useContext(ThemeContext);
  if (!ctx) throw new Error('useTheme must be used inside <ThemeProvider>');
  return ctx;
}

/**
 * Creates a hook that returns StyleSheet styles for the current theme.
 * Styles are built once per color scheme and cached.
 *
 *   const useStyles = createStyles((c) => ({ title: { color: c.text } }));
 */
export function createStyles<S extends StyleInput>(factory: (colors: Colors) => S & StyleInput) {
  const cache = new Map<ColorScheme, Readonly<S>>();
  return function useStyles(): Readonly<S> {
    const { scheme, colors } = useTheme();
    let styles = cache.get(scheme);
    if (!styles) {
      styles = StyleSheet.create(factory(colors));
      cache.set(scheme, styles);
    }
    return styles;
  };
}

type StyleInput = Parameters<typeof StyleSheet.create>[0];
