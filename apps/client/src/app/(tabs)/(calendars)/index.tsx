import { router } from 'expo-router';
import { View } from 'react-native';

import { roleLabel, useCalendars } from '@/api/calendars';
import type { CalendarSummary } from '@/api/client';
import { useNextEntry } from '@/api/entries';
import { AvatarStack } from '@/components/AvatarStack';
import { Button } from '@/components/Button';
import { EmptyState } from '@/components/EmptyState';
import { ErrorState, RefreshError } from '@/components/ErrorState';
import { Icon } from '@/components/Icon';
import { IconButton } from '@/components/IconButton';
import { Page } from '@/components/Page';
import { SkeletonList } from '@/components/Skeleton';
import { Springy } from '@/components/Springy';
import { Text } from '@/components/Text';
import { formatShortDay } from '@/lib/dates';
import { useRefreshOnFocus } from '@/lib/freshness';
import { createStyles, useTheme } from '@/theme/ThemeProvider';
import { radius, space } from '@/theme/tokens';

export default function CalendarsScreen() {
  const calendars = useCalendars();
  const create = () => router.push('/new-calendar');
  const hasCalendars = (calendars.data?.length ?? 0) > 0;

  return (
    <Page
      title="Calendars"
      action={hasCalendars ? <IconButton icon="add" label="New calendar" accent onPress={create} /> : null}
    >
      <CalendarList onCreate={create} />
    </Page>
  );
}

function CalendarList({ onCreate }: { onCreate: () => void }) {
  const styles = useStyles();
  const calendars = useCalendars();
  useRefreshOnFocus(calendars);

  if (calendars.isPending) return <SkeletonList rows={3} sublabel />;
  if (calendars.data === undefined) {
    return <ErrorState error={calendars.error} onRetry={calendars.refetch} retrying={calendars.isFetching} />;
  }
  if (calendars.data.length === 0) {
    return (
      <EmptyState
        icon="calendar-outline"
        title="No calendars yet"
        message="Create a calendar and invite your friends to plan things together."
        action={<Button title="New calendar" onPress={onCreate} />}
      />
    );
  }

  return (
    <>
      {calendars.isRefetchError ? (
        <RefreshError error={calendars.error} onRetry={calendars.refetch} retrying={calendars.isFetching} />
      ) : null}
      <View style={styles.list}>
        {calendars.data.map((c) => (
          <CalendarCard key={c.id} calendar={c} />
        ))}
      </View>
    </>
  );
}

/** A calendar as a soft card: name, members, the next plan's date and a stack of faces. */
function CalendarCard({ calendar }: { calendar: CalendarSummary }) {
  const styles = useStyles();
  const { colors } = useTheme();
  // One request per calendar, so these are not polled; they refresh when the list is shown again.
  const next = useNextEntry(calendar.id, { poll: false });
  useRefreshOnFocus(next);
  const members = `${calendar.memberCount === 1 ? 'Only you' : `${calendar.memberCount} members`} · ${roleLabel[calendar.role]}`;
  const upcoming = next.data ? `${formatShortDay(next.data.date)} · ${next.data.title}` : null;

  return (
    <Springy
      role="link"
      aria-label={[calendar.name, members, upcoming ? `Next: ${upcoming}` : null].filter(Boolean).join(', ')}
      onPress={() => router.push({ pathname: '/calendars/[id]', params: { id: calendar.id } })}
      scaleTo={0.98}
      style={({ pressed, hovered }: { pressed: boolean; hovered?: boolean }) => [
        styles.card,
        (pressed || hovered) && styles.cardPressed,
      ]}
    >
      <View style={styles.cardText}>
        <Text variant="cardTitle" numberOfLines={1}>
          {calendar.name}
        </Text>
        <Text variant="footnote" tone="secondary" numberOfLines={1}>
          {members}
        </Text>
        {next.data !== undefined ? (
          <View style={styles.next}>
            <Icon name="sparkles" size={14} color={upcoming ? colors.accent : colors.textTertiary} />
            <Text variant="footnote" tone={upcoming ? 'primary' : 'tertiary'} numberOfLines={1} style={styles.nextText}>
              {upcoming ?? 'Nothing planned yet'}
            </Text>
          </View>
        ) : null}
      </View>
      <AvatarStack people={calendar.members} total={calendar.memberCount} />
      <Icon name="chevron-forward" size={18} color={colors.textTertiary} />
    </Springy>
  );
}

const useStyles = createStyles((c) => ({
  list: {
    marginTop: space.lg,
    gap: space.md,
  },
  card: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: space.md,
    backgroundColor: c.surface,
    borderRadius: radius.lg,
    paddingVertical: space.lg,
    paddingLeft: space.lg + space.xxs,
    paddingRight: space.md,
  },
  cardPressed: {
    backgroundColor: c.fill,
  },
  cardText: {
    flex: 1,
    gap: 2,
  },
  next: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: space.xs,
    marginTop: space.xs,
  },
  nextText: {
    flexShrink: 1,
    fontWeight: '700',
  },
}));
