/**
 * Colors for both themes. Kept free of React Native imports so
 * `npm run check:contrast` can check them with plain Node; every text and
 * background pair the app uses has to pass WCAG AA there.
 *
 * Golden Hour with a slight 1970s finish: brown-black ink instead of black,
 * burnt-orange coral, tangerine and mustard, on white (light) or a warm
 * dark brown (dark).
 */

export type ColorScheme = 'light' | 'dark';

export type Colors = {
  /** Page background. */
  background: string;
  /** Soft cards: grouped lists, entry rows, banners, round icon buttons. */
  surface: string;
  /** Quiet fills on a card or the page: pressed rows, inputs, segmented control track. */
  fill: string;
  text: string;
  textSecondary: string;
  textTertiary: string;
  hairline: string;
  /** Coral: primary buttons, links, today. */
  accent: string;
  accentPressed: string;
  /** The darker bottom edge under chunky accent buttons. */
  accentEdge: string;
  /** Pale accent tint for selected states and icon badges. */
  accentSoft: string;
  /** Text and icons on top of `accent`. */
  onAccent: string;
  /** The bottom edge under chunky quiet buttons. */
  fillEdge: string;
  danger: string;
  /** The bottom edge under chunky destructive buttons. */
  dangerEdge: string;
  success: string;
  /** The sunset stripes: coral, tangerine and mustard, for markers and arcs. */
  sunset: readonly [string, string, string];
  /** Text on mustard chips and other light sunset fills. */
  onSunset: string;
};

const ink = '#2B1D14';
const mustard = '#F2B33D';
const tangerine = '#F28C38';

export const palette: Record<ColorScheme, Colors> = {
  light: {
    background: '#FFFFFF',
    surface: '#F8F0E6',
    fill: '#F0E4D4',
    text: ink,
    textSecondary: '#65564A',
    textTertiary: '#6E5E52',
    hairline: 'rgba(43, 29, 20, 0.12)',
    accent: '#BC3E1C',
    accentPressed: '#A23517',
    accentEdge: '#8A2D14',
    accentSoft: '#FDEDE6',
    onAccent: '#FFFFFF',
    fillEdge: '#DCC8B1',
    danger: '#B02A22',
    dangerEdge: '#761B16',
    success: '#3D7036',
    sunset: ['#DC4E2A', tangerine, mustard],
    onSunset: ink,
  },
  dark: {
    background: '#1D1611',
    surface: '#2A2019',
    fill: '#3A2D23',
    text: '#F6EBDD',
    textSecondary: '#C9B7A4',
    textTertiary: '#B3A08D',
    hairline: 'rgba(246, 235, 221, 0.14)',
    accent: '#F27A52',
    accentPressed: '#F59272',
    accentEdge: '#A8452A',
    accentSoft: '#4A2619',
    onAccent: '#1D1611',
    fillEdge: '#120D0A',
    danger: '#F2877C',
    dangerEdge: '#A3453B',
    success: '#8FC487',
    sunset: ['#E8603A', tangerine, mustard],
    onSunset: ink,
  },
};

/**
 * Colors users can pick for calendar entries. They are content, not UI chrome:
 * dusty 70s tones that sit with the sunset palette on both light and dark pages.
 * The ids stay as they were; the API stores them.
 */
export const entryColors = [
  { id: 'fjord', value: '#6FA5BD' },
  { id: 'sage', value: '#94B386' },
  { id: 'clay', value: '#E58457' },
  { id: 'rose', value: '#E59393' },
  { id: 'plum', value: '#B08DBA' },
  { id: 'sand', value: '#EDB64C' },
  { id: 'moss', value: '#A9AE5A' },
  { id: 'slate', value: '#A89A8D' },
] as const;

export type EntryColorId = (typeof entryColors)[number]['id'];

/** Text and icons drawn on an entry color, like avatar initials: brown-black ink reads on all eight in both themes. */
export const entryInk = ink;

/** The color value of an entry color id. */
export function entryColor(id: string): string {
  return entryColors.find((c) => c.id === id)?.value ?? entryColors[0].value;
}
