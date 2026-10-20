import { Platform, StyleSheet } from 'react-native';
import { Easing } from 'react-native-reanimated';

/**
 * Design tokens. Every color, size and timing in the app comes from here.
 * Direction: minimal, Apple HIG-like whitespace, hairlines, one accent (Fjord).
 */

export type ColorScheme = 'light' | 'dark';

export type Colors = {
  /** Page background. */
  background: string;
  /** Raised surfaces: grouped lists, sheets, the tab bar. */
  surface: string;
  /** Quiet fills: segmented control track, pressed rows, inputs. */
  fill: string;
  text: string;
  textSecondary: string;
  textTertiary: string;
  hairline: string;
  accent: string;
  accentPressed: string;
  /** Pale accent tint for selected states and icon badges. */
  accentSoft: string;
  /** Text and icons on top of `accent`. */
  onAccent: string;
  danger: string;
  success: string;
};

export const palette: Record<ColorScheme, Colors> = {
  light: {
    background: '#FAF9F7',
    surface: '#FFFFFF',
    fill: '#F1EFEB',
    text: '#1A1918',
    textSecondary: '#6B6862',
    textTertiary: '#9C9890',
    hairline: 'rgba(26, 25, 24, 0.12)',
    accent: '#2E78A6',
    accentPressed: '#266790',
    accentSoft: '#E4EFF6',
    onAccent: '#FFFFFF',
    danger: '#C2453D',
    success: '#3F8A5C',
  },
  dark: {
    background: '#1A1918',
    surface: '#242321',
    fill: '#2F2E2B',
    text: '#EDEBE7',
    textSecondary: '#A8A49D',
    textTertiary: '#77736C',
    hairline: 'rgba(237, 235, 231, 0.14)',
    accent: '#63AEDB',
    accentPressed: '#7DBCE3',
    accentSoft: '#1D3341',
    onAccent: '#0E1A22',
    danger: '#E5776F',
    success: '#74B98E',
  },
};

/**
 * Colors users can pick for calendar entries. They are content, not UI chrome,
 * so they stay muted and work on both light and dark backgrounds.
 */
export const entryColors = [
  { id: 'fjord', value: '#5B9BC4' },
  { id: 'sage', value: '#7FA383' },
  { id: 'clay', value: '#C98B5A' },
  { id: 'rose', value: '#C7798A' },
  { id: 'plum', value: '#9480B8' },
  { id: 'sand', value: '#C4A55A' },
  { id: 'moss', value: '#8C9A5B' },
  { id: 'slate', value: '#8A8F98' },
] as const;

export type EntryColorId = (typeof entryColors)[number]['id'];

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
} as const;

export const layout = {
  /** From this width on, the tab bar turns into a sidebar. */
  wideBreakpoint: 768,
  sidebarWidth: 248,
  /** Readable content width on large screens. */
  contentMaxWidth: 720,
  tabBarHeight: 52,
} as const;
