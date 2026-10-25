import { useId, type ReactNode } from 'react';
import { Image, StyleSheet, View, type StyleProp, type ViewStyle } from 'react-native';
import Svg, { Circle, Defs, Ellipse, LinearGradient, Rect, Stop } from 'react-native-svg';

import { mix } from '@/theme/tokens';

type CoverProps = {
  /** The entry color the sunset is mixed from. */
  color: string;
  style?: StyleProp<ViewStyle>;
  /** Where the sun sets, from 0 (left) to 1 (right); keeps it out from behind a title. */
  sunAt?: number;
  /** Shown on top, e.g. the title; position it inside the cover yourself. */
  children?: ReactNode;
};

const grain = require('../../assets/grain.png');

// The scene is drawn on a 400 × 240 canvas and scaled to cover any box.
const HORIZON = 170;
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
 * A sunset made from an entry's color: a dusky sky, a striped sun on the
 * horizon, hills and water, with fine film grain and a dark fade at the
 * bottom so white text on it stays readable. Stands in for an entry's
 * cover until entries can pick one.
 */
export function Cover({ color, style, sunAt = 0.5, children }: CoverProps) {
  // Gradient ids are global on web; two covers on one page must not share them.
  const id = useId().replace(/[^a-zA-Z0-9]/g, '');
  const sky = [mix(color, '#4A2433', 0.45), color, mix(color, '#FFD9A0', 0.6)];
  const ground = [mix(color, '#2B1D14', 0.35), mix(color, '#2B1D14', 0.6)];
  const sun = '#FFF0C8';
  const cx = 400 * sunAt;

  return (
    <View style={[styles.root, style]}>
      <Svg width="100%" height="100%" viewBox="0 0 400 240" preserveAspectRatio="xMidYMid slice" style={styles.fill}>
        <Defs>
          <LinearGradient id={`sky${id}`} x1="0" y1="0" x2="0" y2={HORIZON} gradientUnits="userSpaceOnUse">
            <Stop offset="0" stopColor={sky[0]} />
            <Stop offset="0.55" stopColor={sky[1]} />
            <Stop offset="1" stopColor={sky[2]} />
          </LinearGradient>
          <LinearGradient id={`ground${id}`} x1="0" y1={HORIZON} x2="0" y2="240" gradientUnits="userSpaceOnUse">
            <Stop offset="0" stopColor={ground[0]} />
            <Stop offset="1" stopColor={ground[1]} />
          </LinearGradient>
          <LinearGradient id={`fade${id}`} x1="0" y1="0" x2="0" y2="1">
            <Stop offset="0" stopColor="#140C08" stopOpacity="0.12" />
            <Stop offset="0.35" stopColor="#140C08" stopOpacity="0" />
            <Stop offset="0.55" stopColor="#140C08" stopOpacity="0" />
            <Stop offset="1" stopColor="#140C08" stopOpacity="0.62" />
          </LinearGradient>
        </Defs>
        <Rect x="0" y="0" width="400" height={HORIZON} fill={`url(#sky${id})`} />
        <Circle cx={cx} cy={HORIZON} r={SUN_R + 26} fill={sun} opacity={0.18} />
        <Circle cx={cx} cy={HORIZON} r={SUN_R} fill={sun} />
        {SUN_GAPS.map(([y, h]) => (
          <Rect key={y} x={cx - SUN_R} y={y} width={SUN_R * 2} height={h} fill={`url(#sky${id})`} />
        ))}
        <Ellipse cx="40" cy={HORIZON + 6} rx="150" ry="34" fill={mix(color, '#2B1D14', 0.25)} />
        <Ellipse cx="380" cy={HORIZON + 4} rx="120" ry="24" fill={mix(color, '#2B1D14', 0.42)} />
        <Rect x="0" y={HORIZON} width="400" height={240 - HORIZON} fill={`url(#ground${id})`} />
        {GLINTS.map(([y, w]) => (
          <Rect key={y} x={cx - w / 2} y={y} width={w} height="3" rx="1.5" fill={sun} opacity={0.5} />
        ))}
        <Rect x="0" y="0" width="400" height="240" fill={`url(#fade${id})`} />
      </Svg>
      <Image source={grain} resizeMode="repeat" style={[styles.fill, styles.grain]} aria-hidden />
      {children}
    </View>
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
