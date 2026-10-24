import { Platform, StyleSheet } from 'react-native';
import { Easing } from 'react-native-reanimated';

/**
 * Design tokens. Every color, size and timing in the app comes from here.
 * Direction: Golden Hour with a slight 1970s finish. A calm, roomy layout on
 * soft rounded cards; warmth comes from the palette, a soft serif for
 * titles (Fraunces) and chunky buttons.
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
  sm: 10,
  md: 16,
  lg: 22,
  xl: 28,
  pill: 999,
} as const;

export const hairline = StyleSheet.hairlineWidth;

/**
 * Font families, loaded with expo-font in the root layout (see fonts.ts).
 * fontWeight picks the face. On web the system font follows in the stack,
 * so text still shows if a font file fails to load; native falls back to
 * the system font by itself.
 */
const systemStack = '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, system-ui, sans-serif';
export const fontFamilies = {
  /** Body text: Manrope, 400–800. */
  body: Platform.select({ web: `Manrope, ${systemStack}`, default: 'Manrope' }),
  /** Titles: Fraunces with its soft, wonky shapes, 600 and 700. */
  display: Platform.select({ web: `Fraunces, Georgia, ${systemStack}`, default: 'Fraunces' }),
};

const body = fontFamilies.body;
const display = fontFamilies.display;

/** Type scale: Fraunces titles over Manrope text. */
export const type = {
  largeTitle: { fontFamily: display, fontSize: 34, lineHeight: 40, fontWeight: '700', letterSpacing: -0.3 },
  title: { fontFamily: display, fontSize: 24, lineHeight: 30, fontWeight: '700', letterSpacing: -0.2 },
  /** Titles of cards and rows that carry a plan, e.g. an entry in the day list. */
  cardTitle: { fontFamily: display, fontSize: 18, lineHeight: 23, fontWeight: '600', letterSpacing: -0.1 },
  headline: { fontFamily: body, fontSize: 16, lineHeight: 22, fontWeight: '700', letterSpacing: -0.1 },
  body: { fontFamily: body, fontSize: 16, lineHeight: 22, fontWeight: '500', letterSpacing: -0.1 },
  callout: { fontFamily: body, fontSize: 15, lineHeight: 21, fontWeight: '500', letterSpacing: -0.1 },
  subhead: { fontFamily: body, fontSize: 14, lineHeight: 20, fontWeight: '500', letterSpacing: 0 },
  footnote: { fontFamily: body, fontSize: 13, lineHeight: 18, fontWeight: '600', letterSpacing: 0 },
  caption: { fontFamily: body, fontSize: 11, lineHeight: 14, fontWeight: '700', letterSpacing: 0.3 },
} as const;

export type TypeVariant = keyof typeof type;

export const motion = {
  duration: { fast: 150, base: 220, slow: 320 },
  /** Calm deceleration, close to the iOS default curve. */
  easing: Easing.bezier(0.2, 0, 0, 1),
  /** Sheets rising into place: settles quickly with a hint of overshoot. */
  spring: { damping: 22, stiffness: 240, mass: 1 },
  /** Springy presses: buttons and cards squash a little and bounce back. */
  press: { damping: 15, stiffness: 420, mass: 0.6 },
  /** How far a pressed card or icon button shrinks. */
  pressScale: 0.96,
} as const;

export const layout = {
  /** From this width on, the tab bar turns into a sidebar. */
  wideBreakpoint: 768,
  sidebarWidth: 248,
  /** Readable content width on large screens. */
  contentMaxWidth: 720,
  tabBarHeight: 60,
} as const;
