import { useState } from 'react';
import { View } from 'react-native';

import { errorMessage, type FriendshipStatus, type Person } from '@/api/client';
import {
  searchPattern,
  useAcceptFriendRequest,
  useDeleteFriendRequest,
  useFriends,
  useRemoveFriend,
  useSendFriendRequest,
  useUserSearch,
} from '@/api/friends';
import { Avatar } from '@/components/Avatar';
import { Button } from '@/components/Button';
import { EmptyState } from '@/components/EmptyState';
import { IconButton } from '@/components/IconButton';
import { ListRow, ListSection } from '@/components/ListSection';
import { Page } from '@/components/Page';
import { Text } from '@/components/Text';
import { TextField } from '@/components/TextField';
import { confirm } from '@/lib/confirm';
import { monthYear, timeAgo } from '@/lib/format';
import { createStyles } from '@/theme/ThemeProvider';
import { space } from '@/theme/tokens';

const AVATAR_SIZE = 36;
// Separators start where the text does, after the avatar.
const SEPARATOR_INSET = space.lg + AVATAR_SIZE + space.md;

export default function FriendsScreen() {
  const styles = useStyles();
  const [search, setSearch] = useState('');
  const searching = search.trim() !== '';

  return (
    <Page title="Friends">
      <View style={styles.search}>
        <TextField
          label="Add a friend"
          value={search}
          onChangeText={setSearch}
          placeholder="Search by username"
          autoCapitalize="none"
          autoCorrect={false}
          autoComplete="off"
          inputMode="search"
          returnKeyType="search"
          maxLength={21}
          clearButtonMode="while-editing"
        />
      </View>
      {searching ? <SearchResults text={search} /> : <FriendLists />}
    </Page>
  );
}

function PersonAvatar({ person }: { person: Person }) {
  return <Avatar id={person.id} name={person.username} url={person.avatarUrl} size={AVATAR_SIZE} />;
}

function SearchResults({ text }: { text: string }) {
  const styles = useStyles();
  const search = useUserSearch(text);
  const send = useSendFriendRequest();
  const accept = useAcceptFriendRequest();
  const busy = (id: string) =>
    (send.isPending && send.variables === id) || (accept.isPending && accept.variables === id);

  if (!searchPattern.test(search.query)) {
    return (
      <Text variant="footnote" tone="tertiary" style={styles.note}>
        {search.query.length < 2 ? 'Type at least 2 characters.' : 'Usernames only have letters, numbers, _ and .'}
      </Text>
    );
  }

  const results = search.data ?? [];
  const actionError = send.error ?? accept.error;
  const footer = search.error ? errorMessage(search.error) : actionError ? errorMessage(actionError) : undefined;

  const action = (person: Person, status: FriendshipStatus) => {
    switch (status) {
      case 'none':
        return <Button variant="plain" title="Add" onPress={() => send.mutate(person.id)} disabled={busy(person.id)} />;
      case 'incoming':
        return (
          <Button size="small" title="Accept" onPress={() => accept.mutate(person.id)} disabled={busy(person.id)} />
        );
      case 'outgoing':
        return <Text tone="tertiary">Requested</Text>;
      case 'friends':
        return <Text tone="tertiary">Friends</Text>;
    }
  };

  return (
    <ListSection title="People" footer={footer} separatorInset={SEPARATOR_INSET}>
      {search.isPending && !search.error ? <ListRow label="Searching…" /> : null}
      {!search.isPending && search.settled && results.length === 0 ? (
        <ListRow
          label={`No one found for @${search.query}`}
          sublabel="Check the spelling, or ask them for their username."
        />
      ) : null}
      {results.map(({ user, friendship }) => (
        <ListRow
          key={user.id}
          leading={<PersonAvatar person={user} />}
          label={`@${user.username}`}
          sublabel={friendship === 'incoming' ? 'Wants to be your friend' : undefined}
          detail={action(user, friendship)}
        />
      ))}
    </ListSection>
  );
}

function FriendLists() {
  const styles = useStyles();
  const friends = useFriends();
  const accept = useAcceptFriendRequest();
  const decline = useDeleteFriendRequest();
  const remove = useRemoveFriend();

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
        <Text tone="secondary" style={styles.errorText}>
          {errorMessage(friends.error)}
        </Text>
        <Button variant="secondary" title="Try again" onPress={() => friends.refetch()} />
      </View>
    );
  }

  const { friends: list, incoming, outgoing } = friends.data;
  if (list.length === 0 && incoming.length === 0 && outgoing.length === 0) {
    return (
      <EmptyState
        icon="people-outline"
        title="No friends yet"
        message="Search for a friend's username above to send them a request."
      />
    );
  }

  const actionError = accept.error ?? decline.error ?? remove.error;
  const pending = (id: string) =>
    (accept.isPending && accept.variables === id) ||
    (decline.isPending && decline.variables === id) ||
    (remove.isPending && remove.variables === id);

  const onRemove = async (person: Person) => {
    const ok = await confirm({
      title: `Remove @${person.username}?`,
      message: 'You can send them a new request later.',
      confirmLabel: 'Remove',
      destructive: true,
    });
    if (ok) remove.mutate(person.id);
  };

  return (
    <>
      {actionError ? (
        <Text variant="footnote" tone="danger" style={styles.note} aria-live="polite">
          {errorMessage(actionError)}
        </Text>
      ) : null}

      {incoming.length > 0 ? (
        <ListSection title="Requests" separatorInset={SEPARATOR_INSET}>
          {incoming.map(({ user, createdAt }) => (
            <ListRow
              key={user.id}
              leading={<PersonAvatar person={user} />}
              label={`@${user.username}`}
              sublabel={timeAgo(createdAt)}
              detail={
                <View style={styles.actions}>
                  <IconButton
                    icon="close"
                    label={`Decline @${user.username}`}
                    onPress={() => decline.mutate(user.id)}
                    disabled={pending(user.id)}
                  />
                  <Button
                    size="small"
                    title="Accept"
                    onPress={() => accept.mutate(user.id)}
                    disabled={pending(user.id)}
                  />
                </View>
              }
            />
          ))}
        </ListSection>
      ) : null}

      {list.length > 0 ? (
        <ListSection title="Friends" separatorInset={SEPARATOR_INSET}>
          {list.map(({ user, since }) => (
            <ListRow
              key={user.id}
              leading={<PersonAvatar person={user} />}
              label={`@${user.username}`}
              sublabel={`Since ${monthYear(since)}`}
              detail={
                <Button
                  variant="plain"
                  title="Remove"
                  destructive
                  onPress={() => onRemove(user)}
                  disabled={pending(user.id)}
                />
              }
            />
          ))}
        </ListSection>
      ) : null}

      {outgoing.length > 0 ? (
        <ListSection title="Sent" separatorInset={SEPARATOR_INSET}>
          {outgoing.map(({ user, createdAt }) => (
            <ListRow
              key={user.id}
              leading={<PersonAvatar person={user} />}
              label={`@${user.username}`}
              sublabel={`Sent ${timeAgo(createdAt)}`}
              detail={
                <Button
                  variant="plain"
                  title="Cancel"
                  onPress={() => decline.mutate(user.id)}
                  disabled={pending(user.id)}
                />
              }
            />
          ))}
        </ListSection>
      ) : null}
    </>
  );
}

const useStyles = createStyles(() => ({
  search: {
    marginTop: space.lg,
  },
  note: {
    marginTop: space.lg,
    paddingHorizontal: space.xs,
  },
  actions: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: space.sm,
  },
  error: {
    alignItems: 'center',
    gap: space.lg,
    paddingVertical: space.xxxl,
  },
  errorText: {
    textAlign: 'center',
  },
}));
