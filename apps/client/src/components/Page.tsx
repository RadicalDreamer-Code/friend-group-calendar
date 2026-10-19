import { useFocusEffect } from 'expo-router';
import { useCallback, type ReactNode } from 'react';
import { useWindowDimensions, View } from 'react-native';
import Animated, {
  Extrapolation,
  interpolate,
  useAnimatedScrollHandler,
  useAnimatedStyle,
  useReducedMotion,
  useSharedValue,
  withTiming,
} from 'react-native-reanimated';
import { useSafeAreaInsets } from 'react-native-safe-area-context';

import { createStyles } from '@/theme/ThemeProvider';
import { hairline, layout, motion, space } from '@/theme/tokens';

import { Text } from './Text';

type PageProps = {
  title: string;
  children: ReactNode;
};

const COMPACT_HEADER_HEIGHT = 44;

/**
 * A scrolling screen with an iOS-style large title. Once the large title
 * scrolls away, a compact title with a hairline fades in at the top.
 * Content eases in each time the screen gains focus.
 */
export function Page({ title, children }: PageProps) {
  const styles = useStyles();
  const insets = useSafeAreaInsets();
  const { width } = useWindowDimensions();
  const wide = width >= layout.wideBreakpoint;
  const reduceMotion = useReducedMotion();

  const scrollY = useSharedValue(0);
  const onScroll = useAnimatedScrollHandler((e) => {
    scrollY.value = e.contentOffset.y;
  });
  const compactHeaderStyle = useAnimatedStyle(() => ({
    opacity: interpolate(scrollY.value, [24, 44], [0, 1], Extrapolation.CLAMP),
  }));

  const entrance = useSharedValue(1);
  useFocusEffect(
    useCallback(() => {
      if (reduceMotion) return;
      entrance.value = 0;
      entrance.value = withTiming(1, { duration: motion.duration.slow, easing: motion.easing });
    }, [reduceMotion, entrance]),
  );
  const entranceStyle = useAnimatedStyle(() => ({
    opacity: entrance.value,
    transform: [{ translateY: (1 - entrance.value) * 8 }],
  }));

  return (
    <View style={styles.root}>
      <Animated.View
        pointerEvents="none"
        style={[
          styles.compactHeader,
          { paddingTop: insets.top, height: insets.top + COMPACT_HEADER_HEIGHT },
          compactHeaderStyle,
        ]}
      >
        <Text variant="headline">{title}</Text>
      </Animated.View>
      <Animated.ScrollView
        onScroll={onScroll}
        scrollEventThrottle={16}
        contentContainerStyle={[
          styles.scrollContent,
          {
            paddingTop: insets.top + (wide ? space.xxxl : space.xl),
            paddingHorizontal: wide ? space.xxl : space.lg,
          },
        ]}
      >
        <Animated.View style={[styles.inner, entranceStyle]}>
          <Text variant="largeTitle" role="heading" aria-level={1} style={styles.title}>
            {title}
          </Text>
          {children}
        </Animated.View>
      </Animated.ScrollView>
    </View>
  );
}

const useStyles = createStyles((c) => ({
  root: {
    flex: 1,
    backgroundColor: c.background,
  },
  compactHeader: {
    position: 'absolute',
    top: 0,
    left: 0,
    right: 0,
    zIndex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: c.background,
    borderBottomWidth: hairline,
    borderBottomColor: c.hairline,
  },
  scrollContent: {
    paddingBottom: space.xxxl,
  },
  inner: {
    width: '100%',
    maxWidth: layout.contentMaxWidth,
    alignSelf: 'center',
  },
  title: {
    marginBottom: space.sm,
  },
}));
