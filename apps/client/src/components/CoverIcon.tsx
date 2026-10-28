import type { ReactNode } from 'react';
import Svg, { Circle, Path, Rect } from 'react-native-svg';

import type { EntryCover } from '@/api/client';

/** A round hole of radius r at (x, y), as a subpath for even-odd fills. */
const hole = (x: number, y: number, r: number) =>
  `M${x - r} ${y}a${r} ${r} 0 1 0 ${r * 2} 0a${r} ${r} 0 1 0 ${-r * 2} 0`;

/** The cover scenes as small flat glyphs on a 24 × 24 grid, in one color. */
const glyphs: Record<EntryCover, (color: string) => ReactNode> = {
  // The striped sun setting over two lines of water.
  lake: (c) => (
    <>
      <Path d="M5 13a7 7 0 0 1 14 0Z" fill={c} />
      <Rect x="2" y="15" width="20" height="2.2" rx="1.1" fill={c} />
      <Rect x="6" y="19" width="12" height="2.2" rx="1.1" fill={c} />
    </>
  ),
  mountains: (c) => (
    <>
      <Path d="M1.5 21 9 7.5l4.2 7 2.8-4.5 6.5 11Z" fill={c} />
      <Circle cx="18.5" cy="4.5" r="2.5" fill={c} />
    </>
  ),
  // A beach umbrella in the sand.
  beach: (c) => (
    <>
      <Path d="M2.5 11.5a9.5 7.5 0 0 1 19 0Z" fill={c} />
      <Rect x="11" y="10" width="2" height="10" fill={c} />
      <Rect x="3" y="19.5" width="18" height="2.2" rx="1.1" fill={c} />
    </>
  ),
  city: (c) => (
    <Path
      d={'M2.5 21.5V10h5.5v11.5Z M9 21.5V3.5h6v18Z M16 21.5V12.5h5.5v9Z' + hole(12, 7, 1) + hole(12, 11, 1)}
      fill={c}
      fillRule="evenodd"
    />
  ),
  // A cocktail glass.
  night_out: (c) => (
    <>
      <Path d="M3.5 3.5h17L13.2 12v7h3.8v2H7v-2h3.8v-7Z" fill={c} />
      <Circle cx="18.5" cy="2.5" r="1.8" fill={c} />
    </>
  ),
  // Fork and knife.
  dinner: (c) => (
    <>
      <Path
        d="M5 2.5h1.4V8h1V2.5h1.4V8h1V2.5h1.4v6.3a2.6 2.6 0 0 1-2.1 2.5V21.5H7.2V11.3A2.6 2.6 0 0 1 5 8.8Z"
        fill={c}
      />
      <Path d="M15.5 2.5c3 .8 4 5 3.2 9.5h-1.4v9.5h-2.2v-19Z" fill={c} />
    </>
  ),
  // A die showing five.
  game_night: (c) => (
    <Path
      d={
        'M7 2.5h10a4.5 4.5 0 0 1 4.5 4.5v10a4.5 4.5 0 0 1-4.5 4.5H7A4.5 4.5 0 0 1 2.5 17V7A4.5 4.5 0 0 1 7 2.5Z' +
        hole(8, 8, 1.8) +
        hole(16, 8, 1.8) +
        hole(12, 12, 1.8) +
        hole(8, 16, 1.8) +
        hole(16, 16, 1.8)
      }
      fill={c}
      fillRule="evenodd"
    />
  ),
};

type CoverIconProps = {
  cover: EntryCover;
  size?: number;
  color: string;
};

/** The small glyph that matches a cover scene, for list rows and day marks. */
export function CoverIcon({ cover, size = 22, color }: CoverIconProps) {
  return (
    <Svg width={size} height={size} viewBox="0 0 24 24" aria-hidden>
      {glyphs[cover](color)}
    </Svg>
  );
}
