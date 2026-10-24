import { useEffect, useState } from 'react';
import { Pressable, View, type LayoutChangeEvent } from 'react-native';
import Animated, { useAnimatedStyle, useReducedMotion, useSharedValue, withTiming } from 'react-native-reanimated';

import { haptics } from '@/lib/haptics';
import { createStyles } from '@/theme/ThemeProvider';
import { hairline, motion, radius, space } from '@/theme/tokens';

import { Text } from './Text';

type Option<T extends string> = { value: T; label: string };

type SegmentedControlProps<T extends string> = {
  options: readonly Option<T>[];
  value: T;
  onChange: (value: T) => void;
  accessibilityLabel: string;
};

const PADDING = 2;

export function SegmentedControl<T extends string>({
  options,
  value,
  onChange,
  accessibilityLabel,
}: SegmentedControlProps<T>) {
  const styles = useStyles();
  const reduceMotion = useReducedMotion();
  const [trackWidth, setTrackWidth] = useState(0);
  const segmentWidth = trackWidth > 0 ? (trackWidth - PADDING * 2) / options.length : 0;
  const index = Math.max(
    0,
    options.findIndex((o) => o.value === value),
  );

  const offset = useSharedValue(0);
  useEffect(() => {
    const target = index * segmentWidth;
    offset.value = reduceMotion
      ? target
      : withTiming(target, { duration: motion.duration.base, easing: motion.easing });
  }, [index, segmentWidth, reduceMotion, offset]);

  const thumbStyle = useAnimatedStyle(() => ({ transform: [{ translateX: offset.value }] }));

  const onLayout = (e: LayoutChangeEvent) => setTrackWidth(e.nativeEvent.layout.width);

  return (
    <View style={styles.track} onLayout={onLayout} role="radiogroup" aria-label={accessibilityLabel}>
      {segmentWidth > 0 ? <Animated.View style={[styles.thumb, { width: segmentWidth }, thumbStyle]} /> : null}
      {options.map((option) => {
        const selected = option.value === value;
        return (
          <Pressable
            key={option.value}
            style={styles.segment}
            onPress={() => {
              if (selected) return;
              haptics.selection();
              onChange(option.value);
            }}
            role="radio"
            aria-checked={selected}
          >
            <Text variant="subhead" tone={selected ? 'primary' : 'secondary'} style={selected && styles.selectedLabel}>
              {option.label}
            </Text>
          </Pressable>
        );
      })}
    </View>
  );
}

const useStyles = createStyles((c) => ({
  track: {
    flexDirection: 'row',
    backgroundColor: c.fill,
    borderRadius: radius.sm + 1,
    padding: PADDING,
    height: 34,
  },
  thumb: {
    position: 'absolute',
    top: PADDING,
    bottom: PADDING,
    left: PADDING,
    borderRadius: radius.sm - 1,
    backgroundColor: c.surface,
    borderWidth: hairline,
    borderColor: c.hairline,
  },
  segment: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    paddingHorizontal: space.sm,
  },
  selectedLabel: {
    fontWeight: '600',
  },
}));
