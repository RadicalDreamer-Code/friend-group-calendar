import { router, useLocalSearchParams } from 'expo-router';
import { useMemo } from 'react';
import { View } from 'react-native';

import { canEditEntries, useCalendar } from '@/api/calendars';
import { type Calendar, type Entry } from '@/api/client';
import { replyOf, useMonthEntries } from '@/api/entries';
import { useUser } from '@/auth/session';
import { CalendarError } from '@/components/CalendarError';
import { ErrorState, RefreshError } from '@/components/ErrorState';
import { IconButton } from '@/components/IconButton';
import { Icon } from '@/components/Icon';
import { MonthCalendar } from '@/components/MonthCalendar';
import { NextUpCard } from '@/components/NextUpCard';
import { Page } from '@/components/Page';
import { SkeletonList } from '@/components/Skeleton';
import { Springy } from '@/components/Springy';
import { Text } from '@/components/Text';
import { formatDay, inDays, isISODate, monthOf, relativeDay, today } from '@/lib/dates';
import { useRefreshOnFocus } from '@/lib/freshness';
import { goBack } from '@/lib/navigation';
import { createStyles, useTheme } from '@/theme/ThemeProvider';
import { entryColor, entryInk, radius, space } from '@/theme/tokens';

/** A calendar's month view: the grid, and the selected day's entries below it. */
export default function MonthScreen() {
  const { id, day } = useLocalSearchParams<{ id: string; day?: string }>();
  // The selected day lives in the URL, so reloading or sharing keeps it.
  const selected = isISODate(day) ? day : today();
  const calendar = useCalendar(id);
  const back = { label: 'Calendars', onPress: () => goBack('/') };
  useRefreshOnFocus(calendar);

  if (calendar.isPending) {
    return (
      <Page title="Calendar" back={back}>
        <SkeletonList rows={4} />
      </Page>
    );
  }
  if (calendar.data === undefined) {
    return (
      <Page title="Calendar" back={back}>
        <CalendarError error={calendar.error} onRetry={calendar.refetch} retrying={calendar.isFetching} />
      </Page>
    );
  }
  return <Month calendar={calendar.data} calendarQuery={calendar} selected={selected} back={back} />;
}

function Month({
  calendar,
  calendarQuery,
  selected,
  back,
}: {
  calendar: Calendar;
  /** For refresh errors; the calendar was loaded before. */
  calendarQuery: ReturnType<typeof useCalendar>;
  selected: string;
  back: { label: string; onPress: () => void };
}) {
  const styles = useStyles();
  const month = monthOf(selected);
  const entries = useMonthEntries(calendar.id, month);
  const canEdit = canEditEntries(calendar.role);
  useRefreshOnFocus(entries);

  const select = (date: string) => router.setParams({ day: date });
  // A new month starts on today when it is that month, else on the 1st.
  const changeMonth = (next: string) => select(monthOf(today()) === next ? today() : `${next}-01`);
  const addEntry = () => router.push({ pathname: '/edit-entry', params: { calendarId: calendar.id, date: selected } });

  const dots = useMemo(() => {
    const byDay: Record<string, string[]> = {};
    for (const e of entries.data ?? []) (byDay[e.date] ??= []).push(entryColor(e.color));
    return byDay;
  }, [entries.data]);

  // The month grid stays usable with what was loaded before; only a first load that fails replaces the day list.
  const refreshError = calendarQuery.isRefetchError
    ? calendarQuery.error
    : entries.isRefetchError
      ? entries.error
      : null;
  const retryRefresh = () => {
    if (calendarQuery.isError) calendarQuery.refetch();
    if (entries.isError) entries.refetch();
  };

  return (
    <Page
      title={calendar.name}
      back={back}
      action={
        <View style={styles.actions}>
          <IconButton
            icon="settings-outline"
            label="Calendar settings"
            onPress={() => router.push({ pathname: '/calendars/[id]/settings', params: { id: calendar.id } })}
          />
          {canEdit ? <IconButton icon="add" label="New entry" accent onPress={addEntry} /> : null}
        </View>
      }
    >
      {refreshError ? (
        <RefreshError
          error={refreshError}
          onRetry={retryRefresh}
          retrying={calendarQuery.isFetching || entries.isFetching}
        />
      ) : null}
      <NextUpCard calendar={calendar} />
      <View style={styles.calendar}>
        <MonthCalendar month={month} onMonthChange={changeMonth} selected={selected} onSelect={select} dots={dots} />
      </View>
      <DayEntries
        calendar={calendar}
        date={selected}
        entries={entries.data?.filter((e) => e.date === selected)}
        error={entries.isError ? entries.error : null}
        onRetry={entries.refetch}
        retrying={entries.isFetching}
        onAdd={canEdit ? addEntry : undefined}
      />
    </Page>
  );
}

function DayEntries({
  calendar,
  date,
  entries,
  error,
  onRetry,
  retrying,
  onAdd,
}: {
  calendar: Calendar;
  date: string;
  entries: Entry[] | undefined;
  error: unknown;
  onRetry: () => void;
  retrying: boolean;
  onAdd?: () => void;
}) {
  const styles = useStyles();
  const me = useUser();
  const day = formatDay(date);
  const relative = relativeDay(date);
  const title = relative === day ? day : `${relative} · ${day}`;

  if (error && !entries) return <ErrorState compact error={error} onRetry={onRetry} retrying={retrying} />;
  if (!entries) return <SkeletonList title={title} rows={1} leading="swatch" sublabel />;
  return (
    <View style={styles.day}>
      <Text variant="caption" tone="secondary" role="heading" aria-level={2} style={styles.dayTitle}>
        {title}
      </Text>
      {entries.map((e) => (
        <EntryCard key={e.id} calendarId={calendar.id} entry={e} me={me.id} />
      ))}
      {entries.length === 0 || onAdd ? <AddCard empty={entries.length === 0} onAdd={onAdd} /> : null}
    </View>
  );
}

/** One plan on the selected day: a soft card with its color tile, title and countdown. */
function EntryCard({ calendarId, entry, me }: { calendarId: string; entry: Entry; me: string }) {
  const styles = useStyles();
  const { colors } = useTheme();
  const summary = `${capitalize(inDays(entry.date))} · ${answerSummary(entry, me)}`;
  return (
    <Springy
      role="button"
      aria-label={`${entry.title}, ${summary}`}
      onPress={() =>
        router.push({ pathname: '/calendars/[id]/entries/[entryId]', params: { id: calendarId, entryId: entry.id } })
      }
      scaleTo={0.98}
      style={({ pressed, hovered }: { pressed: boolean; hovered?: boolean }) => [
        styles.card,
        (pressed || hovered) && styles.cardPressed,
      ]}
    >
      <View style={[styles.tile, { backgroundColor: entryColor(entry.color) }]} aria-hidden>
        <Icon name="sparkles" size={20} color={entryInk} />
      </View>
      <View style={styles.cardText}>
        <Text variant="cardTitle" numberOfLines={1}>
          {entry.title}
        </Text>
        <Text variant="footnote" tone="secondary" numberOfLines={1}>
          {summary}
        </Text>
      </View>
      <Icon name="chevron-forward" size={18} color={colors.textTertiary} />
    </Springy>
  );
}

/** "Nothing planned yet", and for editors a way to add a plan to the day. */
function AddCard({ empty, onAdd }: { empty: boolean; onAdd?: () => void }) {
  const styles = useStyles();
  const { colors } = useTheme();
  const content = (
    <>
      {empty ? (
        <Text tone="secondary" style={styles.addText}>
          Nothing planned yet
        </Text>
      ) : null}
      {onAdd ? (
        <View style={styles.addAction}>
          <Icon name="add" size={18} color={colors.text} />
          <Text variant="headline">Add plan</Text>
        </View>
      ) : null}
    </>
  );
  if (!onAdd) return <View style={styles.addCard}>{content}</View>;
  // With plans on the day, the card is just the action, centered.
  return (
    <Springy
      role="button"
      aria-label="Add a plan on this day"
      onPress={onAdd}
      scaleTo={0.98}
      style={({ pressed, hovered }: { pressed: boolean; hovered?: boolean }) => [
        styles.addCard,
        !empty && styles.addCardCentered,
        (pressed || hovered) && styles.cardPressed,
      ]}
    >
      {content}
    </Springy>
  );
}

const capitalize = (s: string) => s.charAt(0).toUpperCase() + s.slice(1);

/** "You’re going", "You + 2 going", "You declined · 3 going", "3 going", "No answers yet". */
function answerSummary(entry: Entry, userId: string): string {
  const going = entry.accepted.length;
  const mine = replyOf(entry, userId);
  if (mine === 'accepted') return going === 1 ? 'You’re going' : `You + ${going - 1} going`;
  if (mine === 'declined') return going > 0 ? `You declined · ${going} going` : 'You declined';
  return going > 0 ? `${going} going` : 'No answers yet';
}

const TILE = 44;

const useStyles = createStyles((c) => ({
  actions: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: space.sm,
  },
  calendar: {
    marginTop: space.lg,
  },
  day: {
    marginTop: space.lg,
    gap: space.sm,
  },
  dayTitle: {
    paddingHorizontal: space.xs,
    marginBottom: space.xxs,
    letterSpacing: 0.8,
    textTransform: 'uppercase',
  },
  card: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: space.md,
    backgroundColor: c.surface,
    borderRadius: radius.lg,
    paddingVertical: space.sm + 2,
    paddingLeft: space.sm + 2,
    paddingRight: space.md,
  },
  cardPressed: {
    backgroundColor: c.fill,
  },
  tile: {
    width: TILE,
    height: TILE,
    borderRadius: radius.md - 2,
    alignItems: 'center',
    justifyContent: 'center',
  },
  cardText: {
    flex: 1,
    gap: 1,
  },
  addCard: {
    flexDirection: 'row',
    alignItems: 'center',
    minHeight: 52,
    paddingHorizontal: space.lg,
    borderRadius: radius.lg,
    borderWidth: 1.5,
    borderStyle: 'dashed',
    borderColor: c.fillEdge,
  },
  addCardCentered: {
    justifyContent: 'center',
  },
  addText: {
    flex: 1,
  },
  addAction: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: space.xxs,
  },
}));
