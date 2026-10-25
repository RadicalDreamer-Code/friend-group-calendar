import { useEffect } from 'react';
import { View, type DimensionValue, type StyleProp, type ViewStyle } from 'react-native';
import Animated, {
  cancelAnimation,
  useAnimatedStyle,
  useReducedMotion,
  useSharedValue,
  withRepeat,
  withTiming,
  type SharedValue,
} from 'react-native-reanimated';

import { createStyles } from '@/theme/ThemeProvider';
import { radius, space } from '@/theme/tokens';

import { ListRow, ListSection } from './ListSection';

type SkeletonListProps = {
  /** How many placeholder rows. */
  rows?: number;
  /** What sits before the text in the real rows. */
  leading?: 'avatar' | 'swatch';
  /** Whether the real rows have a second line. */
  sublabel?: boolean;
  /** The section title, when the real section has one that is already known. */
  title?: string;
  separatorInset?: number;
};

// Uneven widths read as text rather than as a table of bars.
const LABEL_WIDTHS: DimensionValue[] = ['58%', '42%', '66%', '50%', '38%'];
const SUBLABEL_WIDTHS: DimensionValue[] = ['34%', '46%', '28%', '40%', '36%'];

/**
 * Placeholder rows shaped like the list that is loading, so the page does not
 * jump when the data arrives. They pulse gently, or stay still for people who
 * prefer reduced motion. Screen readers hear "Loading".
 */
export function SkeletonList({ rows = 3, leading, sublabel, title, separatorInset }: SkeletonListProps) {
  const styles = useStyles();
  const pulse = usePulse();
  return (
    <View role="progressbar" aria-label="Loading" aria-busy>
      <ListSection title={title} separatorInset={separatorInset}>
        {Array.from({ length: rows }, (_, i) => (
          <ListRow
            key={i}
            leading={
              leading ? <Bone pulse={pulse} style={leading === 'avatar' ? styles.avatar : styles.swatch} /> : undefined
            }
          >
            <View style={styles.labels} aria-hidden>
              <Bone pulse={pulse} style={[styles.label, { width: LABEL_WIDTHS[i % LABEL_WIDTHS.length] }]} />
              {sublabel ? (
                <Bone pulse={pulse} style={[styles.sublabel, { width: SUBLABEL_WIDTHS[i % SUBLABEL_WIDTHS.length] }]} />
              ) : null}
            </View>
          </ListRow>
        ))}
      </ListSection>
    </View>
  );
}

/** Opacity that breathes between 1 and 0.45, shared by all bones of one skeleton. */
function usePulse(): SharedValue<number> {
  const reduceMotion = useReducedMotion();
  const pulse = useSharedValue(1);
  useEffect(() => {
    if (reduceMotion) return;
    pulse.value = withRepeat(withTiming(0.45, { duration: 900 }), -1, true);
    return () => cancelAnimation(pulse);
  }, [reduceMotion, pulse]);
  return pulse;
}

function Bone({ pulse, style }: { pulse: SharedValue<number>; style: StyleProp<ViewStyle> }) {
  const styles = useStyles();
  const animated = useAnimatedStyle(() => ({ opacity: pulse.value }));
  return <Animated.View style={[styles.bone, style, animated]} />;
}

const AVATAR_SIZE = 36;
const SWATCH_SIZE = 12;

const useStyles = createStyles((c) => ({
  bone: {
    backgroundColor: c.fill,
    borderRadius: radius.pill,
  },
  avatar: {
    width: AVATAR_SIZE,
    height: AVATAR_SIZE,
    borderRadius: radius.pill,
  },
  swatch: {
    width: SWATCH_SIZE,
    height: SWATCH_SIZE,
    borderRadius: radius.pill,
  },
  labels: {
    flex: 1,
    gap: space.sm,
    // The height of a body line, so rows keep their real size.
    paddingVertical: space.xs,
  },
  label: {
    height: 14,
  },
  sublabel: {
    height: 10,
  },
}));
