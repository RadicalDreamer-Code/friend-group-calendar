/**
 * Colors for both themes. Kept free of React Native imports so
 * `npm run check:contrast` can check them with plain Node; every text and
 * background pair the app uses has to pass WCAG AA there.
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
    textSecondary: '#5C5954',
    textTertiary: '#74706A',
    hairline: 'rgba(26, 25, 24, 0.12)',
    accent: '#2E78A6',
    accentPressed: '#266790',
    accentSoft: '#E4EFF6',
    onAccent: '#FFFFFF',
    danger: '#BD433B',
    success: '#377951',
  },
  dark: {
    background: '#1A1918',
    surface: '#242321',
    fill: '#2F2E2B',
    text: '#EDEBE7',
    textSecondary: '#A8A49D',
    textTertiary: '#908C85',
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

/** Text and icons drawn on an entry color, like avatar initials: dark ink reads on all eight in both themes. */
export const entryInk = '#1A1918';

/** The color value of an entry color id. */
export function entryColor(id: string): string {
  return entryColors.find((c) => c.id === id)?.value ?? entryColors[0].value;
}
