import { router } from 'expo-router';

import { roleLabel, useCalendars } from '@/api/calendars';
import { AvatarStack } from '@/components/AvatarStack';
import { Button } from '@/components/Button';
import { EmptyState } from '@/components/EmptyState';
import { ErrorState, RefreshError } from '@/components/ErrorState';
import { IconButton } from '@/components/IconButton';
import { ListRow, ListSection } from '@/components/ListSection';
import { Page } from '@/components/Page';
import { SkeletonList } from '@/components/Skeleton';
import { useRefreshOnFocus } from '@/lib/freshness';

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
      <ListSection>
        {calendars.data.map((c) => (
          <ListRow
            key={c.id}
            label={c.name}
            sublabel={`${c.memberCount === 1 ? 'Only you' : `${c.memberCount} members`} · ${roleLabel[c.role]}`}
            detail={<AvatarStack people={c.members} total={c.memberCount} />}
            chevron
            onPress={() => router.push({ pathname: '/calendars/[id]', params: { id: c.id } })}
          />
        ))}
      </ListSection>
    </>
  );
}
