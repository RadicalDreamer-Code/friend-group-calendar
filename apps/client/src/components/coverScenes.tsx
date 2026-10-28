import type { ReactNode } from 'react';
import { Circle, Ellipse, G, Line, Path, Rect } from 'react-native-svg';

import type { EntryCover } from '@/api/client';
import type { ColorScheme } from '@/theme/tokens';

/**
 * The cover scenes, drawn on the same 400 × 240 canvas as the sunset cover,
 * with the horizon at y = 170. Flat 70s shapes around a striped sun that
 * sets on the horizon; the light versions are a warm late afternoon, the
 * dark ones dusk with a few stars. Keep what matters between x 60–340 and
 * y 80–215: covers crop to fill their box, and the Next up card only shows
 * the bottom strip.
 */

export const W = 400;
export const H = 240;
export const HORIZON = 170;

type SceneProps = {
  /** Fill of the sky gradient, for the gaps through the sun. */
  sky: string;
};

type Scene = {
  /** Top, middle and horizon color of the sky. */
  sky: readonly [string, string, string];
  draw: (props: SceneProps) => ReactNode;
};

const cream = '#FFF0C8';
const ink = '#2B1D14';
const coral = '#DC4E2A';
const tangerine = '#F28C38';
const mustard = '#F2B33D';
const teal = '#3E8A8C';

const SUN_R = 70;
/** Gaps through the lower sun, thicker towards the horizon: [distance above the horizon, height]. */
const SUN_GAPS = [
  [33, 3],
  [23, 4],
  [13, 6],
] as const;

/** The 70s striped sun, half set on the horizon, with a soft halo. */
function Sun({ cx, color, sky, r = SUN_R }: { cx: number; color: string; sky: string; r?: number }) {
  return (
    <>
      <Circle cx={cx} cy={HORIZON} r={r + 26} fill={color} opacity={0.18} />
      <Circle cx={cx} cy={HORIZON} r={r} fill={color} />
      {SUN_GAPS.map(([up, h]) => (
        <Rect key={up} x={cx - r} y={HORIZON - up * (r / SUN_R)} width={r * 2} height={h} fill={sky} />
      ))}
    </>
  );
}

const STARS = [
  [34, 22, 1.4],
  [78, 54, 1],
  [122, 16, 1.6],
  [168, 44, 1],
  [214, 12, 1.2],
  [262, 38, 1.5],
  [306, 18, 1],
  [352, 50, 1.3],
  [382, 14, 1],
  [56, 92, 1],
  [330, 88, 1.1],
] as const;

function Stars({ color = cream }: { color?: string }) {
  return (
    <>
      {STARS.map(([x, y, r]) => (
        <Circle key={x} cx={x} cy={y} r={r} fill={color} opacity={0.85} />
      ))}
    </>
  );
}

/** The sun's reflection on water: [y, width]. */
const GLINTS = [
  [177, 120],
  [188, 88],
  [199, 58],
  [211, 34],
] as const;

function Glints({ cx, color }: { cx: number; color: string }) {
  return (
    <>
      {GLINTS.map(([y, w]) => (
        <Rect key={y} x={cx - w / 2} y={y} width={w} height={3} rx={1.5} fill={color} opacity={0.5} />
      ))}
    </>
  );
}

/** Thin flat lines across water or a table, for the retro print look. */
function Lines({
  from,
  to,
  step,
  color,
  opacity,
}: {
  from: number;
  to: number;
  step: number;
  color: string;
  opacity: number;
}) {
  const ys: number[] = [];
  for (let y = from; y < to; y += step) ys.push(y);
  return (
    <>
      {ys.map((y) => (
        <Rect key={y} x={0} y={y} width={W} height={2} fill={color} opacity={opacity} />
      ))}
    </>
  );
}

const pine = (x: number, y: number, h: number) => `M${x} ${y - h}L${x - h * 0.36} ${y}H${x + h * 0.36}Z`;

/** A point on the quadratic curve p0 → p2 bent towards p1, at t from 0 to 1. */
const onCurve = (p0: number, p1: number, p2: number, t: number) =>
  (1 - t) ** 2 * p0 + 2 * (1 - t) * t * p1 + t ** 2 * p2;

function lake(dark: boolean): Scene {
  const c = dark
    ? { far: '#2A2147', near: '#1A1430', water: '#1B1A40', lines: '#34306A', sun: '#F5C67A', sail: '#D9CBB0' }
    : { far: '#6A6FB0', near: '#45508F', water: '#2C6C97', lines: '#5D9CC0', sun: cream, sail: cream };
  return {
    sky: dark ? ['#1E1A3A', '#5A2F55', '#C7604A'] : ['#E2663F', '#F5A25D', '#FFD9A0'],
    draw: ({ sky }) => (
      <>
        {dark ? <Stars /> : null}
        <Sun cx={250} color={c.sun} sky={sky} />
        <Ellipse cx={40} cy={HORIZON + 6} rx={150} ry={34} fill={c.far} />
        <Ellipse cx={380} cy={HORIZON + 4} rx={120} ry={24} fill={c.near} />
        <Rect x={0} y={HORIZON} width={W} height={H - HORIZON} fill={c.water} />
        <Lines from={182} to={H} step={11} color={c.lines} opacity={0.55} />
        <Glints cx={250} color={c.sun} />
        {/* A small sailboat. */}
        <Path d="M120 126V166H148Z" fill={c.sail} />
        <Path d="M116 130V166H100Z" fill={c.sail} opacity={0.8} />
        <Path d="M96 168H152L144 178H104Z" fill={dark ? '#0F0C1E' : ink} />
      </>
    ),
  };
}

function mountains(dark: boolean): Scene {
  const c = dark
    ? { back: '#2C2547', front: '#1C1733', snow: '#C9C2D9', meadow: '#1C2219', pines: '#10150F', sun: '#F5C67A' }
    : { back: '#8A6A96', front: '#5B4A78', snow: '#FFF6E6', meadow: '#7C8A4A', pines: '#3F5432', sun: tangerine };
  return {
    sky: dark ? ['#18213A', '#3E3557', '#A8594A'] : ['#6FA9B0', '#F2C48D', '#FFE2B0'],
    draw: ({ sky }) => (
      <>
        {dark ? <Stars /> : null}
        <Sun cx={262} color={c.sun} sky={sky} />
        <Path d="M0 170V122L60 104L105 140L165 78L230 150L290 98L345 136L400 92V170Z" fill={c.back} />
        {/* Snow caps that follow the slopes below each peak. */}
        <Path d="M165 78L146.6 97L156 92L165 100L174 92L182 97Z" fill={c.snow} />
        <Path d="M290 98L276.5 111L284 108L291 114L298 107L302.5 108Z" fill={c.snow} />
        <Path d="M0 170V150L52 126L98 160L150 118L214 170Z" fill={c.front} />
        <Path d="M184 170L262 120L318 152L362 128L400 148V170Z" fill={c.front} />
        <Rect x={0} y={HORIZON} width={W} height={H - HORIZON} fill={c.meadow} />
        <Path
          d={[
            pine(44, 204, 30),
            pine(66, 198, 24),
            pine(86, 208, 20),
            pine(318, 206, 26),
            pine(340, 198, 32),
            pine(364, 208, 22),
          ].join('')}
          fill={c.pines}
        />
      </>
    ),
  };
}

function beach(dark: boolean): Scene {
  const c = dark
    ? {
        sea: '#1A2A4A',
        lines: '#2E4670',
        sand: '#5A4636',
        foam: '#C9BFA8',
        leaves: '#14261C',
        trunk: '#2E2219',
        sun: '#F5C67A',
      }
    : {
        sea: '#2F7F8F',
        lines: '#6AB0B8',
        sand: '#EBC489',
        foam: '#FFF8EC',
        leaves: '#3F6B4A',
        trunk: '#7A4E2E',
        sun: '#F59E4A',
      };
  const shore = 'M0 198Q110 186 210 194T400 190';
  return {
    sky: dark ? ['#1B2240', '#47406A', '#D9774E'] : ['#5FA8B8', '#A8D2C8', '#FFE6B5'],
    draw: ({ sky }) => (
      <>
        {dark ? <Stars /> : null}
        <Sun cx={200} color={c.sun} sky={sky} />
        <Rect x={0} y={HORIZON} width={W} height={H - HORIZON} fill={c.sea} />
        <Lines from={176} to={196} step={7} color={c.lines} opacity={0.6} />
        <Glints cx={200} color={c.sun} />
        <Path d={`${shore}V240H0Z`} fill={c.sand} />
        <Path d={shore} stroke={c.foam} strokeWidth={3} fill="none" transform="translate(0 -3)" />
        {/* A palm on the left. */}
        <Path d="M82 214Q72 172 96 132" stroke={c.trunk} strokeWidth={7} strokeLinecap="round" fill="none" />
        <Path
          d="M96 132Q72 114 46 128Q72 122 96 132Z M96 132Q90 104 64 98Q86 110 96 132Z M96 132Q112 104 138 106Q112 114 96 132Z M96 132Q126 124 142 146Q118 132 96 132Z M96 132Q80 140 66 162Q78 138 96 132Z"
          fill={c.leaves}
        />
        {/* A striped beach umbrella on the right. */}
        <G transform="rotate(8 304 170)">
          <Rect x={302} y={128} width={4} height={84} fill={dark ? '#C9BFA8' : ink} />
          <Path d="M262 150A42 24 0 0 1 346 150Z" fill={dark ? '#A8452A' : coral} />
          <Path d="M304 126L290 150H318Z" fill={dark ? '#C99A3A' : mustard} />
        </G>
      </>
    ),
  };
}

/** Buildings: [x, width, top]; they stand on the street at y = 190. */
const CITY_BACK = [
  [0, 40, 120],
  [36, 30, 96],
  [62, 44, 130],
  [100, 26, 84],
  [122, 40, 112],
  [250, 34, 100],
  [280, 46, 124],
  [322, 28, 90],
  [346, 54, 118],
] as const;
const CITY_FRONT = [
  [0, 52, 150],
  [48, 34, 128],
  [78, 46, 158],
  [118, 30, 140],
  [268, 40, 146],
  [304, 34, 124],
  [334, 30, 156],
  [360, 40, 138],
] as const;
const STREET = 190;

/** Window rectangles for a building, a few of them lit, always the same ones. */
function windows(x: number, w: number, top: number, lit: (i: number, j: number) => boolean) {
  const rects: [number, number][] = [];
  for (let row = 0; top + 8 + row * 12 + 6 < STREET - 4; row++) {
    for (let col = 0; x + 6 + col * 10 + 5 < x + w - 4; col++) {
      if (lit(row, col + Math.round(x / 10))) rects.push([x + 6 + col * 10, top + 8 + row * 12]);
    }
  }
  return rects;
}

function city(dark: boolean): Scene {
  const c = dark
    ? { back: '#3A2347', front: '#1E1428', street: '#120C18', window: mustard, sun: '#F5C67A' }
    : { back: '#B5674F', front: '#6B3B3A', street: '#3A2621', window: cream, sun: cream };
  // At dusk most windows are lit; in the afternoon only a few catch the sun.
  const lit = dark
    ? (i: number, j: number) => (i * 7 + j * 3) % 3 !== 0
    : (i: number, j: number) => (i * 7 + j * 3) % 4 === 0;
  return {
    sky: dark ? ['#160F26', '#3B1F45', '#9C3F47'] : ['#EE7F3A', '#F7B267', '#FFE0A8'],
    draw: ({ sky }) => (
      <>
        {dark ? <Stars /> : null}
        <Sun cx={204} color={c.sun} sky={sky} />
        {CITY_BACK.map(([x, w, top]) => (
          <Rect key={x} x={x} y={top} width={w} height={STREET - top} fill={c.back} />
        ))}
        {CITY_FRONT.map(([x, w, top]) => (
          <Rect key={x} x={x} y={top} width={w} height={STREET - top} fill={c.front} />
        ))}
        {CITY_FRONT.flatMap(([x, w, top]) =>
          windows(x, w, top, lit).map(([wx, wy]) => (
            <Rect key={`${wx}-${wy}`} x={wx} y={wy} width={5} height={6} fill={c.window} opacity={dark ? 0.9 : 0.6} />
          )),
        )}
        <Rect x={0} y={STREET} width={W} height={H - STREET} fill={c.street} />
        {[20, 80, 140, 200, 260, 320, 380].map((x) => (
          <Rect key={x} x={x - 14} y={212} width={28} height={3} rx={1.5} fill={mustard} opacity={0.7} />
        ))}
      </>
    ),
  };
}

const BULBS = [mustard, coral, teal, cream];

function nightOut(dark: boolean): Scene {
  const c = dark
    ? { skyline: '#1A0F26', ground: '#120A1A', ball: '#BDB5D3', facets: '#8C82A8', wire: '#4A3A5E', sun: '#F09A8A' }
    : { skyline: '#3B2240', ground: '#2A1730', ball: '#ECE6F4', facets: '#B9B0CC', wire: '#2A1730', sun: '#FFD3C2' };
  // A string of lights sagging across the sky.
  const wire = [0, 48, 200, 120, 400, 48] as const;
  const bulbs = Array.from({ length: 11 }, (_, i) => {
    const t = (i + 0.5) / 11;
    return [onCurve(wire[0], wire[2], wire[4], t), onCurve(wire[1], wire[3], wire[5], t)] as const;
  });
  return {
    sky: dark ? ['#0E0A1E', '#2A1640', '#6E2A55'] : ['#3B2A5C', '#B8487A', '#F59A6B'],
    draw: ({ sky }) => (
      <>
        <Stars />
        <Sun cx={288} r={54} color={c.sun} sky={sky} />
        <Path
          d="M0 170V150H22V138H40V156H62V128H78V146H100V160H128V142H150V170Z M232 170V152H250V134H272V148H296V126H312V144H336V154H360V136H380V150H400V170Z"
          fill={c.skyline}
        />
        <Rect x={0} y={HORIZON} width={W} height={H - HORIZON} fill={c.ground} />
        {/* Reflections of the lights on a wet street. */}
        {[60, 140, 220, 300, 360].map((x, i) => (
          <Rect key={x} x={x - 2} y={184} width={4} height={30} rx={2} fill={BULBS[i % BULBS.length]} opacity={0.35} />
        ))}
        <Path
          d={`M${wire[0]} ${wire[1]}Q${wire[2]} ${wire[3]} ${wire[4]} ${wire[5]}`}
          stroke={c.wire}
          strokeWidth={1.5}
          fill="none"
        />
        {bulbs.map(([x, y], i) => (
          <G key={i}>
            <Circle cx={x} cy={y + 5} r={9} fill={BULBS[i % BULBS.length]} opacity={0.25} />
            <Circle cx={x} cy={y + 5} r={4} fill={BULBS[i % BULBS.length]} />
          </G>
        ))}
        {/* A disco ball with its sparkle. */}
        <Line x1={150} y1={0} x2={150} y2={92} stroke={c.wire} strokeWidth={1.5} />
        <Circle cx={150} cy={118} r={26} fill={c.ball} />
        <Path
          d="M124 118H176M127 105H173M127 131H173M150 92V144M137 95Q130 118 137 141M163 95Q170 118 163 141"
          stroke={c.facets}
          strokeWidth={1.5}
          fill="none"
        />
        <Path d="M108 92l3 8 8 3-8 3-3 8-3-8-8-3 8-3Z M196 140l2 6 6 2-6 2-2 6-2-6-6-2 6-2Z" fill={cream} />
      </>
    ),
  };
}

/** A wine glass standing on the table edge at y = 179, filled a little. */
function Glass({ x, glass, wine }: { x: number; glass: string; wine: string }) {
  const y = 136;
  return (
    <>
      <Path
        d={`M${x - 14} ${y}Q${x - 14} ${y + 22} ${x} ${y + 24}Q${x + 14} ${y + 22} ${x + 14} ${y}Z`}
        fill={glass}
        opacity={0.75}
      />
      <Path
        d={`M${x - 13} ${y + 10}Q${x - 11} ${y + 22} ${x} ${y + 23}Q${x + 11} ${y + 22} ${x + 13} ${y + 10}Z`}
        fill={wine}
      />
      <Rect x={x - 1.5} y={y + 24} width={3} height={18} fill={glass} />
      <Ellipse cx={x} cy={y + 43} rx={10} ry={3} fill={glass} />
    </>
  );
}

function dinner(dark: boolean): Scene {
  const c = dark
    ? {
        far: '#3A2236',
        near: '#2A1828',
        cloth: '#CDB89C',
        check: '#7E3428',
        glass: '#E8DCC8',
        wine: '#6E1E2C',
        plate: '#E0D2BC',
        candle: '#A8452A',
        sun: '#F5C67A',
      }
    : {
        far: '#9A5440',
        near: '#76402F',
        cloth: '#FFF4E0',
        check: coral,
        glass: '#FFF8EC',
        wine: '#9C2A3A',
        plate: '#FFFFFF',
        candle: coral,
        sun: cream,
      };
  const TABLE = 179;
  return {
    sky: dark ? ['#1F1424', '#4E2436', '#A8473A'] : ['#C9553A', '#EE8F55', '#FFD08F'],
    draw: ({ sky }) => (
      <>
        {dark ? <Stars /> : null}
        <Sun cx={200} color={c.sun} sky={sky} />
        <Ellipse cx={60} cy={HORIZON + 8} rx={160} ry={30} fill={c.far} />
        <Ellipse cx={360} cy={HORIZON + 8} rx={140} ry={26} fill={c.near} />
        {/* A gingham tablecloth. */}
        <Rect x={0} y={TABLE} width={W} height={H - TABLE} fill={c.cloth} />
        {Array.from({ length: 17 }, (_, i) => (
          <Rect key={i} x={i * 24} y={TABLE} width={12} height={H - TABLE} fill={c.check} opacity={0.4} />
        ))}
        <Lines from={TABLE + 6} to={H} step={24} color={c.check} opacity={0.4} />
        <Rect x={0} y={TABLE} width={W} height={3} fill={dark ? '#0F0A0E' : ink} opacity={0.25} />
        <Ellipse cx={200} cy={204} rx={50} ry={12} fill={c.plate} />
        <Ellipse cx={200} cy={204} rx={32} ry={7} fill={c.check} opacity={0.18} />
        <Glass x={128} glass={c.glass} wine={c.wine} />
        <Glass x={272} glass={c.glass} wine={c.wine} />
        {/* A candle in the middle. */}
        <Circle cx={200} cy={128} r={20} fill={mustard} opacity={0.3} />
        <Rect x={194} y={140} width={12} height={40} rx={2} fill={c.candle} />
        <Path d="M200 120Q208 131 200 138Q192 131 200 120Z" fill={mustard} />
        <Path d="M200 127Q203 132 200 135Q197 132 200 127Z" fill={cream} />
      </>
    ),
  };
}

/** The pips of a die face on a 0–1 grid. */
const PIPS: Record<3 | 5, readonly (readonly [number, number])[]> = {
  3: [
    [0.25, 0.25],
    [0.5, 0.5],
    [0.75, 0.75],
  ],
  5: [
    [0.25, 0.25],
    [0.75, 0.25],
    [0.5, 0.5],
    [0.25, 0.75],
    [0.75, 0.75],
  ],
};

function Die({
  x,
  y,
  size,
  angle,
  face,
  color,
  pip,
}: {
  x: number;
  y: number;
  size: number;
  angle: number;
  face: 3 | 5;
  color: string;
  pip: string;
}) {
  const cx = x + size / 2;
  const cy = y + size / 2;
  return (
    <G transform={`rotate(${angle} ${cx} ${cy})`}>
      <Rect x={x} y={y + 4} width={size} height={size} rx={size * 0.18} fill={ink} opacity={0.25} />
      <Rect x={x} y={y} width={size} height={size} rx={size * 0.18} fill={color} />
      {PIPS[face].map(([px, py]) => (
        <Circle key={`${px}-${py}`} cx={x + px * size} cy={y + py * size} r={size * 0.09} fill={pip} />
      ))}
    </G>
  );
}

/** The classic meeple, about 30 × 34, standing at (x, y). */
const meeple = (x: number, y: number) =>
  `M${x} ${y - 34}c6 0 9 4 9 9 0 3-2 5-3 6l9 4c3 2 3 6 0 7l-8 1 5 7H${x + 2}l-2-4-2 4H${x - 12}l5-7-8-1c-3-1-3-5 0-7l9-4c-1-1-3-3-3-6 0-5 3-9 9-9Z`;

function gameNight(dark: boolean): Scene {
  const c = dark
    ? { wood: '#4A3026', planks: '#3A241C', die: '#D9C8B0', card: '#D9C8B0', meeple: '#C99A3A', sun: '#F5C67A' }
    : { wood: '#9A6440', planks: '#7E4E31', die: '#FFF4E0', card: '#FFF8EC', meeple: mustard, sun: '#F2A03D' };
  const TABLE = 172;
  return {
    sky: dark ? ['#121A26', '#24384A', '#7A4E4A'] : ['#2F6F73', '#7FA88E', '#F6D49A'],
    draw: ({ sky }) => (
      <>
        {dark ? <Stars /> : null}
        <Sun cx={200} color={c.sun} sky={sky} />
        <Rect x={0} y={TABLE} width={W} height={H - TABLE} fill={c.wood} />
        <Lines from={TABLE + 16} to={H} step={18} color={c.planks} opacity={0.9} />
        <Rect x={0} y={TABLE} width={W} height={3} fill={ink} opacity={0.3} />
        {/* A fan of cards. */}
        {[-22, -6, 10].map((angle, i) => (
          <G key={angle} transform={`rotate(${angle} 320 206)`}>
            <Rect x={304} y={150} width={32} height={46} rx={4} fill={c.card} />
            <Rect
              x={308}
              y={154}
              width={24}
              height={38}
              rx={2}
              fill="none"
              stroke={[coral, teal, tangerine][i]}
              strokeWidth={2}
            />
          </G>
        ))}
        <Path d={meeple(86, 196)} fill={c.meeple} />
        <Die x={126} y={140} size={46} angle={-12} face={5} color={c.die} pip={ink} />
        <Die x={226} y={156} size={36} angle={14} face={3} color={c.die} pip={coral} />
      </>
    ),
  };
}

const scenes: Record<EntryCover, (dark: boolean) => Scene> = {
  lake,
  mountains,
  beach,
  city,
  night_out: nightOut,
  dinner,
  game_night: gameNight,
};

/** The scene for a cover in the given theme. */
export function coverScene(cover: EntryCover, scheme: ColorScheme): Scene {
  return scenes[cover](scheme === 'dark');
}
