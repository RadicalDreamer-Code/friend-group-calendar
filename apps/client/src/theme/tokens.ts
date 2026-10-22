import { Platform, StyleSheet } from 'react-native';
import { Easing } from 'react-native-reanimated';

/**
 * Design tokens. Every color, size and timing in the app comes from here.
 * Direction: minimal, Apple HIG-like whitespace, hairlines, one accent (Fjord).
 */

export { entryColor, entryColors, entryInk, palette } from './colors';
export type { ColorScheme, Colors, EntryColorId } from './colors';

/** 4/8 pt spacing grid. */
export const space = {
  xxs: 2,
  xs: 4,
  sm: 8,
  md: 12,
  lg: 16,
  xl: 24,
  xxl: 32,
  xxxl: 48,
} as const;

export const radius = {
  sm: 8,
  md: 12,
  lg: 16,
  pill: 999,
} as const;

export const hairline = StyleSheet.hairlineWidth;

const fontFamily = Platform.select({
  web: '-apple-system, BlinkMacSystemFont, "SF Pro Text", "Inter", "Segoe UI", Roboto, system-ui, sans-serif',
  default: undefined, // system font: SF Pro on iOS, Roboto on Android
});

/** Type scale modelled on the iOS text styles. */
export const type = {
  largeTitle: { fontFamily, fontSize: 34, lineHeight: 41, fontWeight: '700', letterSpacing: 0.2 },
  title: { fontFamily, fontSize: 22, lineHeight: 28, fontWeight: '600', letterSpacing: -0.2 },
  headline: { fontFamily, fontSize: 17, lineHeight: 22, fontWeight: '600', letterSpacing: -0.4 },
  body: { fontFamily, fontSize: 17, lineHeight: 22, fontWeight: '400', letterSpacing: -0.4 },
  callout: { fontFamily, fontSize: 16, lineHeight: 21, fontWeight: '400', letterSpacing: -0.3 },
  subhead: { fontFamily, fontSize: 15, lineHeight: 20, fontWeight: '400', letterSpacing: -0.2 },
  footnote: { fontFamily, fontSize: 13, lineHeight: 18, fontWeight: '400', letterSpacing: -0.1 },
  caption: { fontFamily, fontSize: 11, lineHeight: 13, fontWeight: '500', letterSpacing: 0.1 },
} as const;

export type TypeVariant = keyof typeof type;

export const motion = {
  duration: { fast: 150, base: 220, slow: 320 },
  /** Calm deceleration, close to the iOS default curve. */
  easing: Easing.bezier(0.2, 0, 0, 1),
  /** Sheets rising into place: settles quickly with a hint of overshoot. */
  spring: { damping: 22, stiffness: 240, mass: 1 },
} as const;

export const layout = {
  /** From this width on, the tab bar turns into a sidebar. */
  wideBreakpoint: 768,
  sidebarWidth: 248,
  /** Readable content width on large screens. */
  contentMaxWidth: 720,
  tabBarHeight: 52,
} as const;
