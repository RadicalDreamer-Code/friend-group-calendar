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
import { ListRow, ListSection } from '@/components/ListSection';
import { MonthCalendar } from '@/components/MonthCalendar';
import { Page } from '@/components/Page';
import { SkeletonList } from '@/components/Skeleton';
import { Text } from '@/components/Text';
import { isISODate, monthOf, relativeDay, today } from '@/lib/dates';
import { useRefreshOnFocus } from '@/lib/freshness';
import { goBack } from '@/lib/navigation';
import { createStyles } from '@/theme/ThemeProvider';
import { entryColor, space } from '@/theme/tokens';

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
  const title = relativeDay(date);

  if (error && !entries) return <ErrorState compact error={error} onRetry={onRetry} retrying={retrying} />;
  if (!entries) return <SkeletonList title={title} rows={1} leading="swatch" sublabel />;
  return (
    <ListSection title={title} separatorInset={SEPARATOR_INSET}>
      {entries.length === 0 ? (
        <ListRow>
          <Text tone="tertiary" style={styles.nothing}>
            Nothing planned
          </Text>
        </ListRow>
      ) : null}
      {entries.map((e) => (
        <ListRow
          key={e.id}
          leading={<View style={[styles.swatch, { backgroundColor: entryColor(e.color) }]} />}
          label={e.title}
          sublabel={answerSummary(e, me.id)}
          chevron
          onPress={() =>
            router.push({
              pathname: '/calendars/[id]/entries/[entryId]',
              params: { id: calendar.id, entryId: e.id },
            })
          }
        />
      ))}
      {onAdd ? <ListRow label="Add entry" action="default" onPress={onAdd} /> : null}
    </ListSection>
  );
}

/** "You’re going · 3 going, 1 declined", "You haven’t answered · 2 going", "No answers yet". */
function answerSummary(entry: Entry, userId: string): string {
  if (entry.accepted.length + entry.declined.length === 0) return 'No answers yet';
  const mine = replyOf(entry, userId);
  const you = mine === 'accepted' ? 'You’re going' : mine === 'declined' ? 'You declined' : 'You haven’t answered';
  const counts = [];
  if (entry.accepted.length > 0) counts.push(`${entry.accepted.length} going`);
  if (entry.declined.length > 0) counts.push(`${entry.declined.length} declined`);
  return `${you} · ${counts.join(', ')}`;
}

const SWATCH_SIZE = 12;
const SEPARATOR_INSET = space.lg + SWATCH_SIZE + space.md;

const useStyles = createStyles(() => ({
  actions: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: space.sm,
  },
  calendar: {
    marginTop: space.lg,
  },
  swatch: {
    width: SWATCH_SIZE,
    height: SWATCH_SIZE,
    borderRadius: SWATCH_SIZE / 2,
  },
  nothing: {
    flex: 1,
  },
}));
