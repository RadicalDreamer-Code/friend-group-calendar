import { ActivityIndicator, Pressable, View, type PressableProps } from 'react-native';
import Animated, {
  useAnimatedStyle,
  useReducedMotion,
  useSharedValue,
  withSpring,
  withTiming,
} from 'react-native-reanimated';

import { haptics } from '@/lib/haptics';
import { createStyles, useTheme } from '@/theme/ThemeProvider';
import { motion, radius, space } from '@/theme/tokens';

import { Text } from './Text';

type ButtonProps = Omit<PressableProps, 'children' | 'style'> & {
  title: string;
  /** primary: chunky coral. secondary: chunky and quiet. plain: text only. */
  variant?: 'primary' | 'secondary' | 'plain';
  destructive?: boolean;
  loading?: boolean;
  /** small: a compact pill for actions inside list rows. */
  size?: 'regular' | 'small';
};

const FACE_HEIGHT = { regular: 50, small: 32 } as const;
/** How far the darker bottom edge shows, and how far the face sinks onto it. */
const EDGE = { regular: 4, small: 3 } as const;

/**
 * A flat, chunky button: the face sits on a darker bottom edge and sinks
 * onto it while pressed (with a light tap on iOS and Android). Plain
 * buttons are just text.
 */
export function Button({
  title,
  variant = 'primary',
  destructive,
  loading,
  disabled,
  size = 'regular',
  onPressIn,
  onPressOut,
  ...props
}: ButtonProps) {
  const styles = useStyles();
  const { colors } = useTheme();
  const reduceMotion = useReducedMotion();
  const inactive = disabled || loading;
  const chunky = variant !== 'plain';
  const edge = chunky ? EDGE[size] : 0;

  const textColor =
    variant === 'primary'
      ? colors.onAccent
      : destructive
        ? colors.danger
        : variant === 'plain'
          ? colors.accent
          : colors.text;
  const [faceColor, edgeColor] =
    variant === 'primary'
      ? destructive
        ? [colors.danger, colors.dangerEdge]
        : [colors.accent, colors.accentEdge]
      : variant === 'secondary'
        ? [colors.surface, colors.fillEdge]
        : ['transparent', 'transparent'];

  const sink = useSharedValue(0);
  const faceStyle = useAnimatedStyle(() => ({ transform: [{ translateY: sink.value }] }));

  return (
    <Pressable
      role="button"
      // The title is replaced by a spinner while loading; the name stays.
      aria-label={title}
      aria-disabled={inactive}
      aria-busy={loading}
      disabled={inactive}
      {...props}
      onPressIn={(e) => {
        if (chunky) {
          haptics.press();
          sink.set(reduceMotion ? edge : withTiming(edge, { duration: motion.duration.fast / 2 }));
        }
        onPressIn?.(e);
      }}
      onPressOut={(e) => {
        if (chunky) sink.set(reduceMotion ? 0 : withSpring(0, motion.press));
        onPressOut?.(e);
      }}
      style={({ pressed }) => [
        styles.root,
        { paddingBottom: edge, borderRadius: size === 'small' ? radius.pill : radius.md },
        !chunky && pressed && styles.plainPressed,
        inactive && !loading && styles.disabled,
      ]}
    >
      {chunky ? (
        <View
          style={[
            styles.edge,
            { top: edge, backgroundColor: edgeColor },
            size === 'small' ? styles.pill : styles.rounded,
          ]}
        />
      ) : null}
      <Animated.View
        style={[
          styles.face,
          { height: variant === 'plain' && size === 'regular' ? 44 : FACE_HEIGHT[size], backgroundColor: faceColor },
          size === 'small' ? [styles.pill, styles.faceSmall] : styles.rounded,
          variant === 'plain' && styles.facePlain,
          faceStyle,
        ]}
      >
        {loading ? (
          <ActivityIndicator color={textColor} />
        ) : (
          <Text variant={size === 'small' ? 'subhead' : 'headline'} style={[{ color: textColor }, styles.label]}>
            {title}
          </Text>
        )}
      </Animated.View>
    </Pressable>
  );
}

const useStyles = createStyles(() => ({
  root: {
    position: 'relative',
  },
  edge: {
    position: 'absolute',
    left: 0,
    right: 0,
    bottom: 0,
  },
  face: {
    alignItems: 'center',
    justifyContent: 'center',
    paddingHorizontal: space.xl,
  },
  faceSmall: {
    paddingHorizontal: space.md + space.xxs,
  },
  facePlain: {
    paddingHorizontal: space.md,
  },
  rounded: {
    borderRadius: radius.md,
  },
  pill: {
    borderRadius: radius.pill,
  },
  label: {
    fontWeight: '700',
  },
  plainPressed: {
    opacity: 0.6,
  },
  disabled: {
    opacity: 0.4,
  },
}));
