import { useLocalSearchParams } from 'expo-router';
import { View } from 'react-native';

import {
  roleDescription,
  roleLabel,
  useCalendar,
  useRemoveMember,
  useTransferOwnership,
  useUpdateMember,
} from '@/api/calendars';
import { errorMessage, type Calendar, type CalendarMember, type MemberRole } from '@/api/client';
import { Avatar } from '@/components/Avatar';
import { CalendarError } from '@/components/CalendarError';
import { EmptyState } from '@/components/EmptyState';
import { ListRow, ListSection } from '@/components/ListSection';
import { Page } from '@/components/Page';
import { SegmentedControl } from '@/components/SegmentedControl';
import { Switch } from '@/components/Switch';
import { Text } from '@/components/Text';
import { confirm } from '@/lib/confirm';
import { monthYear } from '@/lib/format';
import { backTo } from '@/lib/navigation';
import { createStyles } from '@/theme/ThemeProvider';
import { space } from '@/theme/tokens';

const roleOptions = [
  { value: 'editor', label: 'Editor' },
  { value: 'viewer', label: 'Viewer' },
] as const satisfies readonly { value: MemberRole; label: string }[];

/** One member of a calendar, where the owner changes their role and rights. */
export default function MemberScreen() {
  const { id, userId } = useLocalSearchParams<{ id: string; userId: string }>();
  const calendar = useCalendar(id);
  const back = {
    label: 'Settings',
    onPress: () => backTo({ pathname: '/calendars/[id]/settings', params: { id } }),
  };

  if (calendar.isPending) {
    return (
      <Page title="Member" back={back}>
        <ListSection>
          <ListRow label="Loading…" />
        </ListSection>
      </Page>
    );
  }
  if (calendar.isError) {
    return (
      <Page title="Member" back={back}>
        <CalendarError error={calendar.error} onRetry={() => calendar.refetch()} />
      </Page>
    );
  }
  const member = calendar.data.members.find((m) => m.user.id === userId);
  if (!member) {
    return (
      <Page title="Member" back={back}>
        <EmptyState icon="person-outline" title="Not a member" message="They are no longer in this calendar." />
      </Page>
    );
  }
  return (
    <Page title={`@${member.user.username}`} back={back}>
      <MemberDetails calendar={calendar.data} member={member} onDone={back.onPress} />
    </Page>
  );
}

function MemberDetails({
  calendar,
  member,
  onDone,
}: {
  calendar: Calendar;
  member: CalendarMember;
  /** Back to the calendar, once there is nothing left to change here. */
  onDone: () => void;
}) {
  const styles = useStyles();
  const update = useUpdateMember(calendar.id);
  const transfer = useTransferOwnership(calendar.id);
  const remove = useRemoveMember(calendar.id);
  const { user } = member;
  const manageable = calendar.role === 'owner' && member.role !== 'owner';

  // Show the new choice right away instead of waiting for the server.
  const pending = update.isPending ? update.variables : undefined;
  const role = pending?.role ?? member.role;
  const canInvite = pending?.canInvite ?? member.canInvite;
  const error = update.error ?? transfer.error ?? remove.error;

  const onTransfer = async () => {
    const ok = await confirm({
      title: `Make @${user.username} the owner?`,
      message: `@${user.username} will manage the calendar and its members. You stay on as an editor who can invite.`,
      confirmLabel: 'Make owner',
    });
    if (ok) transfer.mutate(user.id, { onSuccess: onDone });
  };

  const onRemove = async () => {
    const ok = await confirm({
      title: `Remove @${user.username}?`,
      message: 'They will no longer see this calendar. You can invite them again later.',
      confirmLabel: 'Remove',
      destructive: true,
    });
    if (ok) remove.mutate(user.id, { onSuccess: onDone });
  };

  return (
    <>
      <View style={styles.header}>
        <Avatar id={user.id} name={user.username} url={user.avatarUrl} size={56} />
        <View style={styles.headerText}>
          <Text variant="headline">{roleLabel[member.role]}</Text>
          <Text variant="subhead" tone="secondary">
            Joined {monthYear(member.joinedAt)}
          </Text>
        </View>
      </View>

      {error ? (
        <Text variant="footnote" tone="danger" style={styles.error} aria-live="polite">
          {errorMessage(error)}
        </Text>
      ) : null}

      {manageable ? (
        <>
          <ListSection title="Role" footer={roleDescription[role === 'owner' ? 'editor' : role]}>
            <ListRow>
              <View style={styles.fill}>
                <SegmentedControl
                  options={roleOptions}
                  value={role === 'owner' ? 'editor' : role}
                  onChange={(next) => update.mutate({ userId: user.id, role: next })}
                  accessibilityLabel="Role"
                />
              </View>
            </ListRow>
          </ListSection>

          <ListSection
            title="Invites"
            footer={
              calendar.membersCanInvite
                ? 'Everyone can invite right now. This matters once you turn that off in the calendar settings.'
                : 'Lets them add their own friends to this calendar.'
            }
          >
            <ListRow
              label="Can invite friends"
              detail={
                <Switch
                  label="Can invite friends"
                  value={canInvite}
                  onValueChange={(next) => update.mutate({ userId: user.id, canInvite: next })}
                />
              }
            />
          </ListSection>

          <ListSection footer="There is one owner. Making someone else the owner lets you leave the calendar.">
            <ListRow label="Make owner" action="default" onPress={onTransfer} disabled={transfer.isPending} />
          </ListSection>

          <ListSection>
            <ListRow label="Remove from calendar" action="destructive" onPress={onRemove} disabled={remove.isPending} />
          </ListSection>
        </>
      ) : null}
    </>
  );
}

const useStyles = createStyles(() => ({
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: space.lg,
    marginTop: space.lg,
  },
  headerText: {
    flex: 1,
    gap: space.xxs,
  },
  error: {
    marginTop: space.lg,
    paddingHorizontal: space.xs,
  },
  fill: {
    flex: 1,
  },
}));
