import { memo, useEffect, useLayoutEffect, useMemo, useRef } from 'react';
import { PanResponder, Pressable, View } from 'react-native';
import Animated, {
  useAnimatedStyle,
  useReducedMotion,
  useSharedValue,
  withSpring,
  withTiming,
} from 'react-native-reanimated';

import {
  addMonths,
  daysOfMonth,
  firstMonth,
  formatDay,
  formatMonth,
  lastMonth,
  leadingBlanks,
  monthOf,
  today as getToday,
  weekdayNames,
} from '@/lib/dates';
import { haptics } from '@/lib/haptics';
import { createStyles, useTheme } from '@/theme/ThemeProvider';
import { motion, radius, space } from '@/theme/tokens';

import { Button } from './Button';
import { IconButton } from './IconButton';
import { Text } from './Text';

/** Most dots one day shows; more entries still show as three. */
const MAX_DOTS = 3;
/** How far a swipe has to travel to change the month. */
const SWIPE_DISTANCE = 40;

type MonthCalendarProps = {
  /** The month shown, YYYY-MM. */
  month: string;
  onMonthChange: (month: string) => void;
  /** The selected day, YYYY-MM-DD. */
  selected: string | null;
  onSelect: (date: string) => void;
  /** Entry colors per day, drawn as dots under the day number. */
  dots?: Record<string, string[]>;
  /** Smaller cells, for picking a date in a form. */
  compact?: boolean;
};

/**
 * A month grid: arrows or a sideways swipe change the month, today is in
 * coral and the selected day is an ink circle that springs in.
 */
export function MonthCalendar({ month, onMonthChange, selected, onSelect, dots, compact }: MonthCalendarProps) {
  const styles = useStyles();
  const today = getToday();
  const canGoBack = month > firstMonth;
  const canGoForward = month < lastMonth;

  // Kept in a ref so the pan responder, created once, sees the current month.
  const go = useRef((_n: number) => {});
  useLayoutEffect(() => {
    go.current = (n: number) => {
      const next = addMonths(month, n);
      if (next < firstMonth || next > lastMonth) return;
      haptics.selection();
      onMonthChange(next);
    };
  });
  const pan = useMemo(
    () =>
      // The handlers only read `go` during a gesture, never while rendering.
      // eslint-disable-next-line react-hooks/refs
      PanResponder.create({
        onMoveShouldSetPanResponder: (_, g) => Math.abs(g.dx) > 12 && Math.abs(g.dx) > Math.abs(g.dy) * 1.5,
        onPanResponderRelease: (_, g) => {
          if (g.dx <= -SWIPE_DISTANCE) go.current(1);
          else if (g.dx >= SWIPE_DISTANCE) go.current(-1);
        },
      }),
    [],
  );

  // The new month slides in from the side it came from.
  const reduceMotion = useReducedMotion();
  const shift = useSharedValue(0);
  const previous = useRef(month);
  useEffect(() => {
    if (previous.current === month) return;
    const direction = month > previous.current ? 1 : -1;
    previous.current = month;
    if (reduceMotion) return;
    shift.value = direction;
    shift.value = withTiming(0, { duration: motion.duration.base, easing: motion.easing });
  }, [month, reduceMotion, shift]);
  const gridStyle = useAnimatedStyle(() => ({
    opacity: 1 - Math.abs(shift.value) * 0.6,
    transform: [{ translateX: shift.value * 24 }],
  }));

  return (
    <View>
      <View style={styles.header}>
        <Text variant={compact ? 'cardTitle' : 'title'} style={styles.monthTitle} aria-live="polite">
          {formatMonth(month)}
        </Text>
        {monthOf(today) !== month ? (
          <Button
            variant="plain"
            title="Today"
            onPress={() => {
              onMonthChange(monthOf(today));
              onSelect(today);
            }}
          />
        ) : null}
        <View style={styles.arrows}>
          <IconButton icon="chevron-back" label="Previous month" onPress={() => go.current(-1)} disabled={!canGoBack} />
          <IconButton
            icon="chevron-forward"
            label="Next month"
            onPress={() => go.current(1)}
            disabled={!canGoForward}
          />
        </View>
      </View>
      <View {...pan.panHandlers}>
        <Weekdays />
        <Animated.View style={gridStyle}>
          <Days
            month={month}
            today={today}
            selected={selected}
            onSelect={onSelect}
            dots={dots}
            compact={compact ?? false}
          />
        </Animated.View>
      </View>
    </View>
  );
}

const Weekdays = memo(function Weekdays() {
  const styles = useStyles();
  return (
    <View style={styles.week} aria-hidden>
      {weekdayNames('short').map((name) => (
        <Text key={name} variant="caption" tone="tertiary" style={styles.weekday}>
          {name.toUpperCase()}
        </Text>
      ))}
    </View>
  );
});

type DaysProps = {
  month: string;
  today: string;
  selected: string | null;
  onSelect: (date: string) => void;
  dots?: Record<string, string[]>;
  compact: boolean;
};

function Days({ month, today, selected, onSelect, dots, compact }: DaysProps) {
  const styles = useStyles();
  const cells: (string | null)[] = [...Array<null>(leadingBlanks(month)).fill(null), ...daysOfMonth(month)];
  while (cells.length % 7 !== 0) cells.push(null);
  const weeks = Array.from({ length: cells.length / 7 }, (_, i) => cells.slice(i * 7, i * 7 + 7));

  return (
    <View>
      {weeks.map((week, i) => (
        <View key={i} style={styles.week}>
          {week.map((date, j) =>
            date ? (
              <Day
                key={date}
                date={date}
                isToday={date === today}
                isSelected={date === selected}
                colors={dots?.[date]}
                compact={compact}
                onSelect={onSelect}
              />
            ) : (
              <View key={`blank-${j}`} style={[styles.cell, compact && styles.cellCompact]} />
            ),
          )}
        </View>
      ))}
    </View>
  );
}

type DayProps = {
  date: string;
  isToday: boolean;
  isSelected: boolean;
  colors?: string[];
  compact: boolean;
  onSelect: (date: string) => void;
};

const Day = memo(function Day({ date, isToday, isSelected, colors: dayColors, compact, onSelect }: DayProps) {
  const styles = useStyles();
  const { colors } = useTheme();
  const count = dayColors?.length ?? 0;

  // The ink circle pops in on the newly selected day, not on a day that is
  // selected when the month appears.
  const reduceMotion = useReducedMotion();
  const pop = useSharedValue(1);
  const wasSelected = useRef(isSelected);
  useEffect(() => {
    if (isSelected && !wasSelected.current && !reduceMotion) {
      pop.set(0.7);
      pop.set(withSpring(1, motion.press));
    }
    wasSelected.current = isSelected;
  }, [isSelected, reduceMotion, pop]);
  const popStyle = useAnimatedStyle(() => ({ transform: [{ scale: pop.value }] }));
  const label = [
    isToday ? 'Today' : null,
    formatDay(date),
    count > 0 ? `${count} ${count === 1 ? 'entry' : 'entries'}` : null,
  ]
    .filter(Boolean)
    .join(', ');

  return (
    <Pressable
      role="button"
      aria-label={label}
      aria-selected={isSelected}
      onPress={() => {
        if (!isSelected) haptics.selection();
        onSelect(date);
      }}
      style={[styles.cell, compact && styles.cellCompact]}
    >
      {({ pressed, hovered }: { pressed: boolean; hovered?: boolean }) => (
        <>
          <Animated.View
            style={[
              styles.number,
              compact && styles.numberCompact,
              (pressed || hovered) && !isSelected && styles.numberPressed,
              isSelected && [styles.numberSelected, popStyle],
            ]}
          >
            <Text
              variant={compact ? 'subhead' : 'callout'}
              style={[
                (isToday || isSelected) && styles.numberEmphasis,
                { color: isSelected ? colors.background : isToday ? colors.accent : colors.text },
              ]}
            >
              {Number(date.slice(8))}
            </Text>
          </Animated.View>
          {compact ? null : (
            <View style={styles.dots}>
              {dayColors?.slice(0, MAX_DOTS).map((c, i) => (
                <View key={i} style={[styles.dot, { backgroundColor: c }]} />
              ))}
            </View>
          )}
        </>
      )}
    </Pressable>
  );
});

const NUMBER_SIZE = 36;
const NUMBER_SIZE_COMPACT = 32;
const DOT_SIZE = 6;

const useStyles = createStyles((c) => ({
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: space.xs,
    marginBottom: space.md,
  },
  monthTitle: {
    flex: 1,
  },
  arrows: {
    flexDirection: 'row',
    gap: space.sm,
  },
  week: {
    flexDirection: 'row',
  },
  weekday: {
    flex: 1,
    textAlign: 'center',
    marginBottom: space.xs,
  },
  cell: {
    flex: 1,
    height: 56,
    alignItems: 'center',
    paddingTop: space.xxs,
  },
  cellCompact: {
    height: 38,
    justifyContent: 'center',
    paddingTop: 0,
  },
  numberPressed: {
    backgroundColor: c.surface,
  },
  numberSelected: {
    backgroundColor: c.text,
  },
  number: {
    width: NUMBER_SIZE,
    height: NUMBER_SIZE,
    borderRadius: radius.pill,
    alignItems: 'center',
    justifyContent: 'center',
  },
  numberCompact: {
    width: NUMBER_SIZE_COMPACT,
    height: NUMBER_SIZE_COMPACT,
  },
  numberEmphasis: {
    fontWeight: '800',
  },
  dots: {
    flexDirection: 'row',
    gap: 3,
    height: DOT_SIZE,
    marginTop: 3,
  },
  dot: {
    width: DOT_SIZE,
    height: DOT_SIZE,
    borderRadius: DOT_SIZE / 2,
  },
}));
