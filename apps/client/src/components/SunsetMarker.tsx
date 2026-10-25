import { useEffect } from 'react';
import { View } from 'react-native';
import Animated, { useAnimatedStyle, useReducedMotion, useSharedValue, withSpring } from 'react-native-reanimated';

import { useTheme } from '@/theme/ThemeProvider';
import { motion } from '@/theme/tokens';

const WIDTH = 22;
const STRIPE = 2;

/**
 * Three thin stripes, coral over tangerine over mustard, like a 70s sunset.
 * Marks the active tab; it springs open when `visible` turns on.
 */
export function SunsetMarker({ visible }: { visible: boolean }) {
  const { colors } = useTheme();
  const reduceMotion = useReducedMotion();
  const open = useSharedValue(visible ? 1 : 0);
  useEffect(() => {
    open.set(reduceMotion ? (visible ? 1 : 0) : withSpring(visible ? 1 : 0, motion.press));
  }, [visible, reduceMotion, open]);
  const style = useAnimatedStyle(() => ({ opacity: Math.min(1, open.value * 2), transform: [{ scaleX: open.value }] }));

  return (
    <Animated.View style={[{ width: WIDTH, borderRadius: 2, overflow: 'hidden' }, style]} aria-hidden>
      {colors.sunset.map((color) => (
        <View key={color} style={{ height: STRIPE, backgroundColor: color }} />
      ))}
    </Animated.View>
  );
}
