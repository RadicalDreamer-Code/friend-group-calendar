import type { StyleProp, ViewStyle } from 'react-native';
import Svg, { Circle } from 'react-native-svg';

import { useTheme } from '@/theme/ThemeProvider';

/**
 * Three concentric sunset rings (mustard inside, then tangerine and coral),
 * meant to peek in from a corner. `size` is the outer diameter.
 */
export function RainbowArc({ size, style }: { size: number; style?: StyleProp<ViewStyle> }) {
  const { colors } = useTheme();
  const band = size * 0.067;
  const inner = size * 0.29;
  const rings = [colors.sunset[2], colors.sunset[1], colors.sunset[0]];
  return (
    <Svg width={size} height={size} viewBox={`0 0 ${size} ${size}`} style={style} aria-hidden>
      {rings.map((color, i) => (
        <Circle
          key={color}
          cx={size / 2}
          cy={size / 2}
          r={inner + band * (i + 0.5)}
          stroke={color}
          strokeWidth={band}
          fill="none"
        />
      ))}
    </Svg>
  );
}
