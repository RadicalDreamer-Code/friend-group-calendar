import { useFocusEffect } from 'expo-router';
import { useCallback, type ReactNode } from 'react';
import { Platform, Pressable, useWindowDimensions, View } from 'react-native';
import Animated, {
  Extrapolation,
  interpolate,
  useAnimatedScrollHandler,
  useAnimatedStyle,
  useReducedMotion,
  useSharedValue,
  withSpring,
  withTiming,
} from 'react-native-reanimated';
import { useSafeAreaInsets } from 'react-native-safe-area-context';

import { createStyles, useTheme } from '@/theme/ThemeProvider';
import { hairline, layout, motion, space } from '@/theme/tokens';

import { Icon } from './Icon';
import { Text } from './Text';

type PageProps = {
  title: string;
  /** A back button above the title, for screens pushed onto a stack. */
  back?: { label: string; onPress: () => void };
  /** Shown next to the large title, e.g. a button that adds something. */
  action?: ReactNode;
  /** A sheet (a modal like "New entry"): on web it rises in on a spring, as native modals do. */
  sheet?: boolean;
  children: ReactNode;
};

const COMPACT_HEADER_HEIGHT = 44;

/**
 * A scrolling screen with an iOS-style large title. Once the large title
 * scrolls away, a compact title with a hairline fades in at the top.
 * Content eases in each time the screen gains focus; sheets rise in on a
 * spring on web, where there is no native modal animation.
 */
export function Page({ title, back, action, sheet, children }: PageProps) {
  const styles = useStyles();
  const { colors } = useTheme();
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

  const rise = sheet && Platform.OS === 'web';
  const entrance = useSharedValue(1);
  useFocusEffect(
    useCallback(() => {
      if (reduceMotion) return;
      entrance.set(0);
      entrance.set(
        rise ? withSpring(1, motion.spring) : withTiming(1, { duration: motion.duration.slow, easing: motion.easing }),
      );
    }, [reduceMotion, entrance, rise]),
  );
  const distance = rise ? 48 : 8;
  const entranceStyle = useAnimatedStyle(() => ({
    opacity: Math.min(1, entrance.value * (rise ? 2 : 1)),
    transform: [{ translateY: (1 - entrance.value) * distance }],
  }));

  return (
    <View style={styles.root}>
      {/* Repeats the title for sighted people once it scrolls away; screen readers have the heading. */}
      <Animated.View
        pointerEvents="none"
        aria-hidden
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
          {back ? (
            <Pressable
              role="link"
              aria-label={`Back to ${back.label}`}
              onPress={back.onPress}
              hitSlop={8}
              style={({ pressed }) => [styles.back, pressed && styles.backPressed]}
            >
              <Icon name="chevron-back" size={20} color={colors.accent} />
              <Text tone="accent" numberOfLines={1}>
                {back.label}
              </Text>
            </Pressable>
          ) : null}
          <View style={styles.titleRow}>
            <Text variant="largeTitle" role="heading" aria-level={1} style={styles.title}>
              {title}
            </Text>
            {action}
          </View>
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
  back: {
    flexDirection: 'row',
    alignItems: 'center',
    alignSelf: 'flex-start',
    gap: space.xxs,
    // Lines the chevron up with the title below.
    marginLeft: -space.xs,
    marginBottom: space.sm,
  },
  backPressed: {
    opacity: 0.6,
  },
  titleRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: space.md,
    marginBottom: space.sm,
  },
  title: {
    flex: 1,
  },
}));
