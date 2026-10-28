import { useId, type ReactNode } from 'react';
import { Image, StyleSheet, View, type StyleProp, type ViewStyle } from 'react-native';
import Svg, { Circle, Defs, Ellipse, LinearGradient, Rect, Stop } from 'react-native-svg';

import type { EntryCover } from '@/api/client';
import { useTheme } from '@/theme/ThemeProvider';
import { mix } from '@/theme/tokens';

import { coverScene, H, HORIZON, W } from './coverScenes';

type CoverProps = {
  /** The entry color the sunset is mixed from when there is no scene. */
  color: string;
  /** The entry's cover scene; without one, a sunset made from `color`. */
  cover?: EntryCover | null;
  style?: StyleProp<ViewStyle>;
  /**
   * Where the sunset's sun sets, from 0 (left) to 1 (right); keeps it out
   * from behind a title. Scenes place their sun themselves.
   */
  sunAt?: number;
  /**
   * Which part of the scene a box with another shape keeps. 'bottom' keeps
   * the horizon, water and dark fade in a wide, short box, for text at the
   * bottom, and leaves out the last of the water so more of the sun shows;
   * 'middle' crops evenly.
   */
  anchor?: 'middle' | 'bottom';
  /** Shown on top, e.g. the title; position it inside the cover yourself. */
  children?: ReactNode;
};

const grain = require('../../assets/grain.png');

/** How much of the canvas an anchor='bottom' cover shows: down to just below the last glint. */
const BOTTOM_VIEW = 218;
const SUN_R = 70;
/** Gaps through the lower sun, thicker towards the horizon: the 70s striped sun. */
const SUN_GAPS = [
  [137, 3],
  [147, 4],
  [157, 6],
] as const;
/** Glints of the sun on the water: [y, width]. */
const GLINTS = [
  [177, 120],
  [188, 88],
  [199, 58],
  [211, 34],
] as const;

/**
 * An entry's cover: its scene (lake, mountains, …) in the light or dark
 * version, or without one a sunset made from the entry's color. Both get
 * fine film grain and a dark fade at the bottom so white text on it stays
 * readable; scenes also get a warm fade. Serves the entry header, the Next
 * up card and the cover picker.
 */
export function Cover({ color, cover, style, sunAt = 0.5, anchor = 'middle', children }: CoverProps) {
  const { scheme } = useTheme();
  // Gradient ids are global on web; two covers on one page must not share them.
  const id = useId().replace(/[^a-zA-Z0-9]/g, '');
  const scene = cover ? coverScene(cover, scheme) : null;
  const sky = scene ? scene.sky : [mix(color, '#4A2433', 0.45), color, mix(color, '#FFD9A0', 0.6)];

  return (
    <View style={[styles.root, style]}>
      <Svg
        width="100%"
        height="100%"
        viewBox={`0 0 ${W} ${anchor === 'bottom' ? BOTTOM_VIEW : H}`}
        preserveAspectRatio={anchor === 'bottom' ? 'xMidYMax slice' : 'xMidYMid slice'}
        style={styles.fill}
      >
        <Defs>
          <LinearGradient id={`sky${id}`} x1="0" y1="0" x2="0" y2={HORIZON} gradientUnits="userSpaceOnUse">
            <Stop offset="0" stopColor={sky[0]} />
            <Stop offset="0.55" stopColor={sky[1]} />
            <Stop offset="1" stopColor={sky[2]} />
          </LinearGradient>
          {scene ? null : (
            <LinearGradient id={`ground${id}`} x1="0" y1={HORIZON} x2="0" y2={H} gradientUnits="userSpaceOnUse">
              <Stop offset="0" stopColor={mix(color, '#2B1D14', 0.35)} />
              <Stop offset="1" stopColor={mix(color, '#2B1D14', 0.6)} />
            </LinearGradient>
          )}
          <LinearGradient id={`fade${id}`} x1="0" y1="0" x2="0" y2="1">
            <Stop offset="0" stopColor="#140C08" stopOpacity="0.12" />
            <Stop offset="0.35" stopColor="#140C08" stopOpacity="0" />
            <Stop offset="0.55" stopColor="#140C08" stopOpacity="0" />
            <Stop offset="1" stopColor="#140C08" stopOpacity="0.62" />
          </LinearGradient>
        </Defs>
        <Rect x="0" y="0" width={W} height={HORIZON} fill={`url(#sky${id})`} />
        {scene ? (
          <>
            {scene.draw({ sky: `url(#sky${id})` })}
            {/* The warm fade of old film. */}
            <Rect x="0" y="0" width={W} height={H} fill="#F28C38" opacity={scheme === 'dark' ? 0.06 : 0.1} />
          </>
        ) : (
          <Sunset color={color} cx={W * sunAt} id={id} />
        )}
        <Rect x="0" y="0" width={W} height={H} fill={`url(#fade${id})`} />
      </Svg>
      <Image source={grain} resizeMode="repeat" style={[styles.fill, styles.grain]} aria-hidden />
      {children}
    </View>
  );
}

/** The cover of an entry without a scene: a striped sun setting over hills and water in the entry's color. */
function Sunset({ color, cx, id }: { color: string; cx: number; id: string }) {
  const sun = '#FFF0C8';
  return (
    <>
      <Circle cx={cx} cy={HORIZON} r={SUN_R + 26} fill={sun} opacity={0.18} />
      <Circle cx={cx} cy={HORIZON} r={SUN_R} fill={sun} />
      {SUN_GAPS.map(([y, h]) => (
        <Rect key={y} x={cx - SUN_R} y={y} width={SUN_R * 2} height={h} fill={`url(#sky${id})`} />
      ))}
      <Ellipse cx="40" cy={HORIZON + 6} rx="150" ry="34" fill={mix(color, '#2B1D14', 0.25)} />
      <Ellipse cx="380" cy={HORIZON + 4} rx="120" ry="24" fill={mix(color, '#2B1D14', 0.42)} />
      <Rect x="0" y={HORIZON} width={W} height={H - HORIZON} fill={`url(#ground${id})`} />
      {GLINTS.map(([y, w]) => (
        <Rect key={y} x={cx - w / 2} y={y} width={w} height="3" rx="1.5" fill={sun} opacity={0.5} />
      ))}
    </>
  );
}

const styles = StyleSheet.create({
  root: {
    overflow: 'hidden',
  },
  fill: {
    position: 'absolute',
    top: 0,
    left: 0,
    right: 0,
    bottom: 0,
  },
  grain: {
    width: '100%',
    height: '100%',
    opacity: 0.35,
  },
});
