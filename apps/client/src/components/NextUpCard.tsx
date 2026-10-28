import { router } from 'expo-router';
import { useEffect } from 'react';
import { useWindowDimensions, View } from 'react-native';
import Animated, {
  cancelAnimation,
  Easing,
  useAnimatedStyle,
  useReducedMotion,
  useSharedValue,
  withRepeat,
  withTiming,
} from 'react-native-reanimated';

import type { Calendar, EntryCover } from '@/api/client';
import { isTrip, useNextEntry } from '@/api/entries';
import { dayOf, formatDay, formatRange, formatShortDay, inDays } from '@/lib/dates';
import { useRefreshOnFocus, useToday } from '@/lib/freshness';
import { createStyles } from '@/theme/ThemeProvider';
import { entryColor, layout, radius, space } from '@/theme/tokens';

import { Cover } from './Cover';
import { CoverIcon } from './CoverIcon';
import { Icon } from './Icon';
import { RainbowArc } from './RainbowArc';
import { Springy } from './Springy';
import { Text } from './Text';

/**
 * Above a calendar's month: the next entry from today on, with its cover,
 * a countdown chip and how many are going; a trip that is still running
 * shows which of its days today is. Opens the entry. Hidden while nothing
 * is coming up, and while it loads or cannot be loaded; the month below
 * shows the same entries either way.
 */
export function NextUpCard({ calendar }: { calendar: Calendar }) {
  const styles = useStyles();
  const { width } = useWindowDimensions();
  const next = useNextEntry(calendar.id);
  const now = useToday();
  useRefreshOnFocus(next);
  const entry = next.data;
  if (!entry) return null;

  const going = entry.accepted.length;
  const members = calendar.members.length;
  const running = entry.date < now;
  const when = running ? dayOf(entry.date, entry.endDate, now) : inDays(entry.date, now);
  const label = running ? `Now · ${when}` : `Next up · ${when}`;
  const dates = isTrip(entry) ? formatRange(entry.date, entry.endDate, now) : formatShortDay(entry.date, now);
  const spokenDates = isTrip(entry) ? dates : formatDay(entry.date, now);

  return (
    <Springy
      role="link"
      aria-label={`${label}: ${entry.title}, ${spokenDates}. ${going} of ${members} going.`}
      onPress={() =>
        router.push({ pathname: '/calendars/[id]/entries/[entryId]', params: { id: calendar.id, entryId: entry.id } })
      }
      containerStyle={styles.shadow}
      scaleTo={0.98}
    >
      <Cover
        color={entryColor(entry.color)}
        cover={entry.cover}
        // Taller on wide screens, so more of the sunset shows in a long card.
        style={[styles.cover, width >= layout.wideBreakpoint && styles.coverWide]}
        sunAt={0.62}
        // A wide card crops the scene to a strip; keep the bottom, where the white text is.
        anchor="bottom"
      >
        <RainbowArc size={ARC} style={styles.arc} />
        <View style={styles.text}>
          <View style={styles.chip}>
            <Text variant="caption" tone="onSunset" style={styles.chipText}>
              {label}
            </Text>
          </View>
          <Text variant="cardTitle" style={styles.title} numberOfLines={1}>
            {entry.title}
          </Text>
          <Text variant="footnote" style={styles.meta} numberOfLines={1}>
            {dates} · {going} of {members} going
          </Text>
        </View>
        <BobbingIcon cover={entry.cover} />
      </Cover>
    </Springy>
  );
}

/** The cover's icon, bobbing gently; still with reduced motion. */
function BobbingIcon({ cover }: { cover: EntryCover | null }) {
  const styles = useStyles();
  const reduceMotion = useReducedMotion();
  const bob = useSharedValue(0);
  useEffect(() => {
    if (reduceMotion) return;
    bob.set(withRepeat(withTiming(1, { duration: 1500, easing: Easing.inOut(Easing.sin) }), -1, true));
    return () => cancelAnimation(bob);
  }, [reduceMotion, bob]);
  const style = useAnimatedStyle(() => ({
    transform: [{ translateY: -4 * bob.value }, { rotate: `${-6 * bob.value}deg` }],
  }));
  return (
    <Animated.View style={[styles.icon, style]} aria-hidden>
      {cover ? <CoverIcon cover={cover} size={30} color={ICON} /> : <Icon name="sparkles" size={30} color={ICON} />}
    </Animated.View>
  );
}

const ARC = 104;
const ICON = '#FFF0C8';
const HEIGHT = 128;

const useStyles = createStyles((c) => ({
  shadow: {
    marginTop: space.lg,
    borderRadius: radius.lg,
    shadowColor: c.accent,
    shadowOpacity: 0.35,
    shadowRadius: 14,
    shadowOffset: { width: 0, height: 10 },
    elevation: 6,
  },
  cover: {
    height: HEIGHT,
    borderRadius: radius.lg,
  },
  coverWide: {
    height: 172,
  },
  arc: {
    position: 'absolute',
    top: -ARC / 3,
    right: -ARC / 3,
  },
  text: {
    position: 'absolute',
    left: space.lg,
    right: 64,
    bottom: space.md + 2,
    alignItems: 'flex-start',
    gap: 3,
  },
  chip: {
    backgroundColor: c.sunset[2],
    borderRadius: radius.pill,
    paddingHorizontal: space.sm + 1,
    paddingVertical: 3,
    marginBottom: 2,
  },
  chipText: {
    fontWeight: '800',
  },
  title: {
    color: '#FFFFFF',
    fontSize: 22,
    lineHeight: 27,
    fontWeight: '700',
  },
  meta: {
    color: '#FFFFFF',
    opacity: 0.92,
  },
  icon: {
    position: 'absolute',
    right: space.lg,
    bottom: space.md,
    shadowColor: '#000000',
    shadowOpacity: 0.35,
    shadowRadius: 6,
    shadowOffset: { width: 0, height: 4 },
  },
}));
