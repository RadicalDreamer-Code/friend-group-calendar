import { Pressable, type PressableProps, type StyleProp, type ViewStyle } from 'react-native';
import Animated, { useAnimatedStyle, useReducedMotion, useSharedValue, withSpring } from 'react-native-reanimated';

import { motion } from '@/theme/tokens';

type SpringyProps = Omit<PressableProps, 'ref'> & {
  /** Style of the outer view that scales, e.g. `flex: 1` for a card that fills a row. */
  containerStyle?: StyleProp<ViewStyle>;
  /** How small it gets while pressed. */
  scaleTo?: number;
};

/**
 * A Pressable that squashes a little while pressed and springs back on
 * release. It stays still for people who prefer reduced motion.
 */
export function Springy({
  containerStyle,
  scaleTo = motion.pressScale,
  onPressIn,
  onPressOut,
  ...props
}: SpringyProps) {
  const reduceMotion = useReducedMotion();
  const scale = useSharedValue(1);
  const animated = useAnimatedStyle(() => ({ transform: [{ scale: scale.value }] }));
  const springTo = (value: number) => {
    if (!reduceMotion) scale.set(withSpring(value, motion.press));
  };

  return (
    <Animated.View style={[containerStyle, animated]}>
      <Pressable
        {...props}
        onPressIn={(e) => {
          springTo(scaleTo);
          onPressIn?.(e);
        }}
        onPressOut={(e) => {
          springTo(1);
          onPressOut?.(e);
        }}
      />
    </Animated.View>
  );
}
