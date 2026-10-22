import { router } from 'expo-router';
import { View } from 'react-native';

import { roleLabel, useCalendars } from '@/api/calendars';
import { errorMessage } from '@/api/client';
import { AvatarStack } from '@/components/AvatarStack';
import { Button } from '@/components/Button';
import { EmptyState } from '@/components/EmptyState';
import { IconButton } from '@/components/IconButton';
import { ListRow, ListSection } from '@/components/ListSection';
import { Page } from '@/components/Page';
import { Text } from '@/components/Text';
import { useRefreshOnFocus } from '@/lib/freshness';
import { createStyles } from '@/theme/ThemeProvider';
import { space } from '@/theme/tokens';

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

  if (calendars.isPending) {
    return (
      <ListSection>
        <ListRow label="Loading…" />
      </ListSection>
    );
  }
  if (calendars.isError) {
    return (
      <View style={styles.error}>
        <Text tone="secondary" style={styles.errorText}>
          {errorMessage(calendars.error)}
        </Text>
        <Button variant="secondary" title="Try again" onPress={() => calendars.refetch()} />
      </View>
    );
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
  );
}

const useStyles = createStyles(() => ({
  error: {
    alignItems: 'center',
    gap: space.lg,
    paddingVertical: space.xxxl,
  },
  errorText: {
    textAlign: 'center',
  },
}));
