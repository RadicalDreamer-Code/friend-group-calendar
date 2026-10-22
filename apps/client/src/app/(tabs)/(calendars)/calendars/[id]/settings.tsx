import { router, useLocalSearchParams } from 'expo-router';
import { useState } from 'react';
import { View } from 'react-native';

import {
  calendarNameMaxLength,
  roleLabel,
  useCalendar,
  useDeleteCalendar,
  useLeaveCalendar,
  useUpdateCalendar,
} from '@/api/calendars';
import { errorMessage, type Calendar, type CalendarMember } from '@/api/client';
import { useUser } from '@/auth/session';
import { Avatar } from '@/components/Avatar';
import { Button } from '@/components/Button';
import { CalendarError } from '@/components/CalendarError';
import { ListRow, ListSection } from '@/components/ListSection';
import { Page } from '@/components/Page';
import { Switch } from '@/components/Switch';
import { TextField } from '@/components/TextField';
import { confirm } from '@/lib/confirm';
import { backTo } from '@/lib/navigation';
import { createStyles } from '@/theme/ThemeProvider';
import { space } from '@/theme/tokens';

const AVATAR_SIZE = 36;
const SEPARATOR_INSET = space.lg + AVATAR_SIZE + space.md;

export default function CalendarSettingsScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const calendar = useCalendar(id);
  const back = {
    label: calendar.data?.name ?? 'Calendar',
    onPress: () => backTo({ pathname: '/calendars/[id]', params: { id } }),
  };

  if (calendar.isPending) {
    return (
      <Page title="Settings" back={back}>
        <ListSection>
          <ListRow label="Loading…" />
        </ListSection>
      </Page>
    );
  }
  if (calendar.isError) {
    return (
      <Page title="Settings" back={back}>
        <CalendarError error={calendar.error} onRetry={() => calendar.refetch()} />
      </Page>
    );
  }
  return (
    <Page title="Settings" back={back}>
      <Settings calendar={calendar.data} />
    </Page>
  );
}

function Settings({ calendar }: { calendar: Calendar }) {
  const owner = calendar.role === 'owner';
  return (
    <>
      {/* Keyed by the name so the field starts over after a save. */}
      {owner ? <NameSection key={calendar.name} calendar={calendar} /> : null}
      <MembersSection calendar={calendar} />
      {owner ? <InviteRightsSection calendar={calendar} /> : null}
      <DangerSection calendar={calendar} />
    </>
  );
}

function NameSection({ calendar }: { calendar: Calendar }) {
  const styles = useStyles();
  const [name, setName] = useState(calendar.name);
  const update = useUpdateCalendar(calendar.id);
  const trimmed = name.trim();
  const changed = trimmed !== '' && trimmed !== calendar.name;

  const save = () => {
    if (changed && !update.isPending) update.mutate({ name: trimmed });
  };

  return (
    <View style={styles.name}>
      <TextField
        label="Name"
        value={name}
        onChangeText={(text) => {
          setName(text);
          if (update.isError) update.reset();
        }}
        onSubmitEditing={save}
        maxLength={calendarNameMaxLength}
        returnKeyType="done"
        error={update.error ? errorMessage(update.error) : null}
      />
      {changed ? (
        <View style={styles.nameActions}>
          <Button variant="plain" title="Cancel" onPress={() => setName(calendar.name)} />
          <Button size="small" title="Save" onPress={save} loading={update.isPending} />
        </View>
      ) : null}
    </View>
  );
}

function memberSublabel(member: CalendarMember, isMe: boolean): string {
  const parts = [roleLabel[member.role]];
  if (member.role !== 'owner' && member.canInvite) parts.push('Can invite');
  if (isMe) parts.unshift('You');
  return parts.join(' · ');
}

function MembersSection({ calendar }: { calendar: Calendar }) {
  const me = useUser();
  const owner = calendar.role === 'owner';
  const count = calendar.members.length;

  return (
    <ListSection
      title={`${count} ${count === 1 ? 'member' : 'members'}`}
      separatorInset={SEPARATOR_INSET}
      footer={calendar.canInvite ? undefined : 'Only the owner and members they allow can invite friends.'}
    >
      {calendar.members.map((m) => {
        const isMe = m.user.id === me.id;
        const manage = owner && !isMe;
        return (
          <ListRow
            key={m.user.id}
            leading={<Avatar id={m.user.id} name={m.user.username} url={m.user.avatarUrl} size={AVATAR_SIZE} />}
            label={`@${m.user.username}`}
            sublabel={memberSublabel(m, isMe)}
            chevron={manage}
            onPress={
              manage
                ? () =>
                    router.push({
                      pathname: '/calendars/[id]/members/[userId]',
                      params: { id: calendar.id, userId: m.user.id },
                    })
                : undefined
            }
          />
        );
      })}
      {calendar.canInvite ? (
        <ListRow
          label="Invite friends"
          action="default"
          chevron
          onPress={() => router.push({ pathname: '/calendars/[id]/invite', params: { id: calendar.id } })}
        />
      ) : null}
    </ListSection>
  );
}

function InviteRightsSection({ calendar }: { calendar: Calendar }) {
  const update = useUpdateCalendar(calendar.id);
  // Show the new position right away instead of waiting for the server.
  const value = update.isPending
    ? (update.variables.membersCanInvite ?? calendar.membersCanInvite)
    : calendar.membersCanInvite;

  return (
    <ListSection
      title="Invites"
      footer={
        update.error
          ? errorMessage(update.error)
          : value
            ? 'Every member can invite their friends.'
            : 'Only you and the members you allow can invite. Open a member to allow them.'
      }
    >
      <ListRow
        label="Everyone can invite"
        detail={
          <Switch
            label="Everyone can invite"
            value={value}
            onValueChange={(membersCanInvite) => update.mutate({ membersCanInvite })}
          />
        }
      />
    </ListSection>
  );
}

function DangerSection({ calendar }: { calendar: Calendar }) {
  const me = useUser();
  const remove = useDeleteCalendar(calendar.id);
  const leave = useLeaveCalendar(calendar.id);
  const owner = calendar.role === 'owner';
  const others = calendar.members.length - 1;
  const error = remove.error ?? leave.error;

  const onDelete = async () => {
    const ok = await confirm({
      title: `Delete “${calendar.name}”?`,
      message:
        others > 0
          ? `The calendar disappears for all ${others + 1} members. This cannot be undone.`
          : 'This cannot be undone.',
      confirmLabel: 'Delete',
      destructive: true,
    });
    if (ok) remove.mutate(undefined, { onSuccess: () => router.dismissTo('/') });
  };

  const onLeave = async () => {
    const ok = await confirm({
      title: `Leave “${calendar.name}”?`,
      message: 'You need a new invite to join again.',
      confirmLabel: 'Leave',
      destructive: true,
    });
    if (ok) leave.mutate(me.id, { onSuccess: () => router.dismissTo('/') });
  };

  const footer = error
    ? errorMessage(error)
    : owner && others > 0
      ? 'To leave, open a member and make them the owner first.'
      : undefined;

  return (
    <ListSection footer={footer}>
      {owner ? (
        <ListRow label="Delete calendar" action="destructive" onPress={onDelete} disabled={remove.isPending} />
      ) : (
        <ListRow label="Leave calendar" action="destructive" onPress={onLeave} disabled={leave.isPending} />
      )}
    </ListSection>
  );
}

const useStyles = createStyles(() => ({
  name: {
    marginTop: space.lg,
  },
  nameActions: {
    flexDirection: 'row',
    justifyContent: 'flex-end',
    alignItems: 'center',
    gap: space.xs,
    marginTop: space.sm,
  },
}));
