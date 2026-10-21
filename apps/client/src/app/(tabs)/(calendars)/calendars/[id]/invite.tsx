import { router, useLocalSearchParams } from 'expo-router';
import { useState } from 'react';
import { View } from 'react-native';

import { roleDescription, useCalendar, useInviteMember } from '@/api/calendars';
import { errorMessage, type Calendar, type MemberRole } from '@/api/client';
import { useFriends } from '@/api/friends';
import { Avatar } from '@/components/Avatar';
import { Button } from '@/components/Button';
import { CalendarError } from '@/components/CalendarError';
import { EmptyState } from '@/components/EmptyState';
import { ListRow, ListSection } from '@/components/ListSection';
import { Page } from '@/components/Page';
import { SegmentedControl } from '@/components/SegmentedControl';
import { Text } from '@/components/Text';
import { goBack } from '@/lib/navigation';
import { createStyles } from '@/theme/ThemeProvider';
import { space } from '@/theme/tokens';

const AVATAR_SIZE = 36;
const SEPARATOR_INSET = space.lg + AVATAR_SIZE + space.md;

const roleOptions = [
  { value: 'viewer', label: 'Viewer' },
  { value: 'editor', label: 'Editor' },
] as const satisfies readonly { value: MemberRole; label: string }[];

export default function InviteScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const calendar = useCalendar(id);
  const back = {
    label: calendar.data?.name ?? 'Calendar',
    onPress: () => goBack({ pathname: '/calendars/[id]/settings', params: { id } }),
  };

  return (
    <Page title="Invite friends" back={back}>
      {calendar.isPending ? (
        <ListSection>
          <ListRow label="Loading…" />
        </ListSection>
      ) : calendar.isError ? (
        <CalendarError error={calendar.error} onRetry={() => calendar.refetch()} />
      ) : calendar.data.canInvite ? (
        <InviteFriends calendar={calendar.data} />
      ) : (
        <EmptyState
          icon="lock-closed-outline"
          title="You cannot invite yet"
          message="Ask the owner of this calendar to let you invite friends."
        />
      )}
    </Page>
  );
}

function InviteFriends({ calendar }: { calendar: Calendar }) {
  const styles = useStyles();
  const friends = useFriends();
  const invite = useInviteMember(calendar.id);
  // Nobody joins with a higher role than the person inviting them.
  const canPickRole = calendar.role !== 'viewer';
  const [role, setRole] = useState<MemberRole>('viewer');
  const members = new Set(calendar.members.map((m) => m.user.id));

  if (friends.isPending) {
    return (
      <ListSection>
        <ListRow label="Loading…" />
      </ListSection>
    );
  }
  if (friends.isError) {
    return (
      <View style={styles.error}>
        <Text tone="secondary" style={styles.center}>
          {errorMessage(friends.error)}
        </Text>
        <Button variant="secondary" title="Try again" onPress={() => friends.refetch()} />
      </View>
    );
  }
  const list = friends.data.friends;
  if (list.length === 0) {
    return (
      <EmptyState
        icon="people-outline"
        title="No friends yet"
        message="You can invite friends to calendars. Add some first."
        action={<Button variant="secondary" title="Find friends" onPress={() => router.navigate('/friends')} />}
      />
    );
  }

  return (
    <>
      <Text variant="subhead" tone="secondary" style={styles.intro}>
        Friends you invite join right away and can leave whenever they like.
      </Text>

      {canPickRole ? (
        <ListSection title="Join as" footer={roleDescription[role]}>
          <ListRow>
            <View style={styles.fill}>
              <SegmentedControl options={roleOptions} value={role} onChange={setRole} accessibilityLabel="Join as" />
            </View>
          </ListRow>
        </ListSection>
      ) : null}

      <ListSection
        title="Friends"
        separatorInset={SEPARATOR_INSET}
        footer={
          invite.error
            ? errorMessage(invite.error)
            : canPickRole
              ? undefined
              : 'They join as viewers: ' + roleDescription.viewer.toLowerCase()
        }
      >
        {list.map(({ user }) => {
          const member = members.has(user.id);
          const busy = invite.isPending && invite.variables.userId === user.id;
          return (
            <ListRow
              key={user.id}
              leading={<Avatar id={user.id} name={user.username} url={user.avatarUrl} size={AVATAR_SIZE} />}
              label={`@${user.username}`}
              detail={
                member ? (
                  <Text tone="tertiary">Member</Text>
                ) : (
                  <Button
                    size="small"
                    title="Invite"
                    onPress={() => invite.mutate({ userId: user.id, role: canPickRole ? role : 'viewer' })}
                    disabled={busy}
                  />
                )
              }
            />
          );
        })}
      </ListSection>
    </>
  );
}

const useStyles = createStyles(() => ({
  intro: {
    marginTop: space.sm,
  },
  fill: {
    flex: 1,
  },
  error: {
    alignItems: 'center',
    gap: space.lg,
    paddingVertical: space.xxxl,
  },
  center: {
    textAlign: 'center',
  },
}));
