import { router, useLocalSearchParams } from 'expo-router';
import { Pressable, View } from 'react-native';

import { canEditEntries, useCalendar } from '@/api/calendars';
import { errorMessage, type Calendar, type CalendarMember, type Entry } from '@/api/client';
import { isEntryGone, replyOf, useEntry, useRespond, type Reply } from '@/api/entries';
import { useUser } from '@/auth/session';
import { Avatar } from '@/components/Avatar';
import { Button } from '@/components/Button';
import { CalendarError } from '@/components/CalendarError';
import { EmptyState } from '@/components/EmptyState';
import { RefreshError } from '@/components/ErrorState';
import { Icon, type IconName } from '@/components/Icon';
import { IconButton } from '@/components/IconButton';
import { ListRow, ListSection } from '@/components/ListSection';
import { Page } from '@/components/Page';
import { SkeletonList } from '@/components/Skeleton';
import { Text } from '@/components/Text';
import { relativeDay } from '@/lib/dates';
import { haptics } from '@/lib/haptics';
import { useRefreshOnFocus } from '@/lib/freshness';
import { backTo } from '@/lib/navigation';
import { createStyles, useTheme } from '@/theme/ThemeProvider';
import { entryColor, radius, space } from '@/theme/tokens';

const AVATAR_SIZE = 32;
const SEPARATOR_INSET = space.lg + AVATAR_SIZE + space.md;

/** One entry: who is going, who declined, who has not answered, and your own answer. */
export default function EntryScreen() {
  const { id, entryId } = useLocalSearchParams<{ id: string; entryId: string }>();
  const calendar = useCalendar(id);
  const entry = useEntry(id, entryId);
  useRefreshOnFocus(calendar, entry);
  const back = {
    label: calendar.data?.name ?? 'Calendar',
    onPress: () => backTo({ pathname: '/calendars/[id]', params: entry.data ? { id, day: entry.data.date } : { id } }),
  };

  const retry = () => {
    if (calendar.isError) calendar.refetch();
    if (entry.isError) entry.refetch();
  };
  const retrying = calendar.isFetching || entry.isFetching;

  if (entry.isError && isEntryGone(entry.error)) {
    return (
      <Page title="Entry" back={back}>
        <EmptyState
          icon="calendar-clear-outline"
          title="Entry deleted"
          message="Someone removed it from the calendar."
          action={
            <Button variant="secondary" title={`Back to ${calendar.data?.name ?? 'calendar'}`} onPress={back.onPress} />
          }
        />
      </Page>
    );
  }
  if ((calendar.isError && !calendar.data) || (entry.isError && !entry.data)) {
    return (
      <Page title="Entry" back={back}>
        <CalendarError error={calendar.error ?? entry.error} onRetry={retry} retrying={retrying} />
      </Page>
    );
  }
  if (!calendar.data || !entry.data) {
    return (
      <Page title="Entry" back={back}>
        <SkeletonList title="Going" rows={2} leading="avatar" separatorInset={SEPARATOR_INSET} />
      </Page>
    );
  }

  const canEdit = canEditEntries(calendar.data.role);
  return (
    <Page
      title={entry.data.title}
      back={back}
      action={
        canEdit ? (
          <IconButton
            icon="create-outline"
            label="Edit entry"
            onPress={() => router.push({ pathname: '/edit-entry', params: { calendarId: id, entryId } })}
          />
        ) : null
      }
    >
      {calendar.isRefetchError || entry.isRefetchError ? (
        <RefreshError error={calendar.error ?? entry.error} onRetry={retry} retrying={retrying} />
      ) : null}
      <Details calendar={calendar.data} entry={entry.data} />
    </Page>
  );
}

function Details({ calendar, entry }: { calendar: Calendar; entry: Entry }) {
  const styles = useStyles();
  const me = useUser();
  const respond = useRespond(calendar.id, entry.id, me.id);
  const mine = replyOf(entry, me.id);

  const byId = new Map(calendar.members.map((m) => [m.user.id, m]));
  const people = (ids: string[]) => ids.map((id) => byId.get(id)).filter((m): m is CalendarMember => !!m);
  const going = people(entry.accepted);
  const declined = people(entry.declined);
  const answered = new Set([...entry.accepted, ...entry.declined]);
  const waiting = calendar.members.filter((m) => !answered.has(m.user.id));

  // Tapping your current answer again takes it back.
  const answer = (reply: Exclude<Reply, null>) => {
    haptics.selection();
    respond.mutate(mine === reply ? null : reply);
  };

  return (
    <>
      <View style={styles.meta}>
        <View style={[styles.swatch, { backgroundColor: entryColor(entry.color) }]} />
        <Text tone="secondary">{relativeDay(entry.date)}</Text>
      </View>
      {entry.createdBy ? (
        <Text variant="footnote" tone="tertiary" style={styles.addedBy}>
          Added by {entry.createdBy.id === me.id ? 'you' : `@${entry.createdBy.username}`}
        </Text>
      ) : null}

      <View style={styles.answers} role="radiogroup" aria-label="Your answer">
        <AnswerButton
          icon="checkmark"
          label={mine === 'accepted' ? 'Going' : 'Accept'}
          selected={mine === 'accepted'}
          onPress={() => answer('accepted')}
        />
        <AnswerButton
          icon="close"
          label={mine === 'declined' ? 'Declined' : 'Decline'}
          selected={mine === 'declined'}
          onPress={() => answer('declined')}
        />
      </View>
      <Text variant="footnote" tone={respond.isError ? 'danger' : 'tertiary'} style={styles.answerHint}>
        {respond.isError
          ? errorMessage(respond.error)
          : mine
            ? 'Tap your answer again to take it back.'
            : 'Let the others know if you’re in.'}
      </Text>

      <People title="Going" members={going} me={me.id} empty="Nobody yet." />
      <People title="Declined" members={declined} me={me.id} />
      <People title="No answer yet" members={waiting} me={me.id} />
    </>
  );
}

function AnswerButton({
  icon,
  label,
  selected,
  onPress,
}: {
  icon: IconName;
  label: string;
  selected: boolean;
  onPress: () => void;
}) {
  const styles = useStyles();
  const { colors } = useTheme();
  const color = selected ? colors.onAccent : colors.text;
  return (
    <Pressable
      role="radio"
      aria-checked={selected}
      aria-label={label}
      onPress={onPress}
      style={({ pressed }) => [styles.answer, selected && styles.answerSelected, pressed && styles.answerPressed]}
    >
      <Icon name={icon} size={20} color={color} />
      <Text variant="headline" style={{ color }}>
        {label}
      </Text>
    </Pressable>
  );
}

function People({
  title,
  members,
  me,
  empty,
}: {
  title: string;
  members: CalendarMember[];
  me: string;
  empty?: string;
}) {
  if (members.length === 0 && !empty) return null;
  return (
    <ListSection title={`${title} · ${members.length}`} separatorInset={SEPARATOR_INSET}>
      {members.length === 0 ? <ListRow label={empty} /> : null}
      {members.map((m) => (
        <ListRow
          key={m.user.id}
          leading={<Avatar id={m.user.id} name={m.user.username} url={m.user.avatarUrl} size={AVATAR_SIZE} />}
          label={`@${m.user.username}`}
          detail={m.user.id === me ? 'You' : undefined}
        />
      ))}
    </ListSection>
  );
}

const SWATCH_SIZE = 12;

const useStyles = createStyles((c) => ({
  meta: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: space.sm,
  },
  swatch: {
    width: SWATCH_SIZE,
    height: SWATCH_SIZE,
    borderRadius: SWATCH_SIZE / 2,
  },
  addedBy: {
    marginTop: space.xs,
  },
  answers: {
    flexDirection: 'row',
    gap: space.md,
    marginTop: space.xl,
  },
  answer: {
    flex: 1,
    height: 50,
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: space.sm,
    borderRadius: radius.md,
    backgroundColor: c.fill,
  },
  answerSelected: {
    backgroundColor: c.accent,
  },
  answerPressed: {
    opacity: 0.7,
  },
  answerHint: {
    marginTop: space.sm,
    paddingHorizontal: space.lg,
  },
}));
