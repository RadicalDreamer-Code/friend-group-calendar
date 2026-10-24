import Constants from 'expo-constants';
import * as ImagePicker from 'expo-image-picker';
import { router } from 'expo-router';
import { ActivityIndicator, Pressable, View } from 'react-native';

import {
  useRemoveAvatar,
  useRevokeOtherSessions,
  useRevokeSession,
  useSessions,
  useSignOut,
  useUploadAvatar,
} from '@/api/account';
import { errorMessage, type Session } from '@/api/client';
import { useServerStatus, type ServerStatus } from '@/api/health';
import { useUser } from '@/auth/session';
import { Avatar } from '@/components/Avatar';
import { Button } from '@/components/Button';
import { ListRow, ListSection } from '@/components/ListSection';
import { Page } from '@/components/Page';
import { SegmentedControl } from '@/components/SegmentedControl';
import { SkeletonList } from '@/components/Skeleton';
import { Text } from '@/components/Text';
import { confirm } from '@/lib/confirm';
import { lastActive, monthYear } from '@/lib/format';
import { createStyles, useTheme, type ThemePreference } from '@/theme/ThemeProvider';
import { radius, space } from '@/theme/tokens';

export { ScreenError as ErrorBoundary } from '@/components/ScreenError';

const themeOptions = [
  { value: 'system', label: 'System' },
  { value: 'light', label: 'Light' },
  { value: 'dark', label: 'Dark' },
] as const satisfies readonly { value: ThemePreference; label: string }[];

const statusLabel: Record<ServerStatus, string> = {
  checking: 'Checking…',
  connected: 'Connected',
  degraded: 'Database unavailable',
  unreachable: 'Unreachable',
};

const AVATAR_SIZE = 72;

export default function ProfileScreen() {
  const styles = useStyles();
  const { colors, preference, setPreference } = useTheme();
  const status = useServerStatus();
  const user = useUser();
  const upload = useUploadAvatar();
  const removeAvatar = useRemoveAvatar();
  const signOut = useSignOut();

  const statusColor = {
    checking: colors.textTertiary,
    connected: colors.success,
    degraded: colors.danger,
    unreachable: colors.danger,
  }[status];

  const displayName = user.username ?? user.email;
  const avatarBusy = upload.isPending || removeAvatar.isPending;
  const avatarError = upload.error ?? removeAvatar.error;

  const pickAvatar = async () => {
    // Web ignores allowsEditing; the API center-crops to a square either way.
    const result = await ImagePicker.launchImageLibraryAsync({
      mediaTypes: ['images'],
      allowsEditing: true,
      aspect: [1, 1],
      quality: 0.9,
    });
    if (result.canceled || !result.assets[0]) return;
    removeAvatar.reset();
    upload.mutate(result.assets[0]);
  };

  const onSignOut = async () => {
    if (await confirm({ title: 'Sign out?', confirmLabel: 'Sign out', destructive: true })) signOut.mutate();
  };

  return (
    <Page title="Profile">
      <View style={styles.header}>
        <Pressable
          onPress={pickAvatar}
          disabled={avatarBusy}
          role="button"
          aria-label={user.avatarUrl ? 'Change profile photo' : 'Add profile photo'}
          style={({ pressed }) => pressed && styles.pressed}
        >
          <Avatar id={user.id} name={displayName} url={user.avatarUrl} size={AVATAR_SIZE} />
          {avatarBusy ? (
            <View style={styles.avatarBusy}>
              <ActivityIndicator color="#FFFFFF" />
            </View>
          ) : null}
        </Pressable>
        <View style={styles.headerText}>
          <Text variant="title" numberOfLines={1}>
            {user.username ? `@${user.username}` : user.email}
          </Text>
          <View style={styles.photoActions}>
            <Button
              variant="plain"
              title={user.avatarUrl ? 'Change photo' : 'Add photo'}
              onPress={pickAvatar}
              disabled={avatarBusy}
            />
            {user.avatarUrl ? (
              <Button
                variant="plain"
                destructive
                title="Remove"
                disabled={avatarBusy}
                onPress={() => {
                  upload.reset();
                  removeAvatar.mutate();
                }}
              />
            ) : null}
          </View>
        </View>
      </View>
      {avatarError ? (
        <Text variant="footnote" tone="danger" style={styles.avatarError} aria-live="polite">
          {errorMessage(avatarError)}
        </Text>
      ) : null}

      <ListSection title="Account">
        <ListRow
          label="Username"
          detail={user.username ? `@${user.username}` : '–'}
          chevron
          onPress={() => router.push('/edit-username')}
        />
        <ListRow label="Email" detail={user.email} />
        <ListRow label="Member since" detail={monthYear(user.createdAt)} />
      </ListSection>

      <DevicesSection />

      <ListSection title="Appearance" footer="System follows the light or dark setting of your device.">
        <ListRow>
          <View style={styles.fill}>
            <SegmentedControl
              options={themeOptions}
              value={preference}
              onChange={setPreference}
              accessibilityLabel="Theme"
            />
          </View>
        </ListRow>
      </ListSection>

      <ListSection title="About">
        <ListRow
          label="Server"
          detail={
            <View style={styles.status}>
              <View style={[styles.statusDot, { backgroundColor: statusColor }]} />
              <Text tone="secondary">{statusLabel[status]}</Text>
            </View>
          }
        />
        <ListRow label="Version" detail={Constants.expoConfig?.version ?? '–'} />
      </ListSection>

      <ListSection>
        <ListRow label="Sign out" action="destructive" onPress={onSignOut} disabled={signOut.isPending} />
      </ListSection>
    </Page>
  );
}

function DevicesSection() {
  const sessions = useSessions();
  const revoke = useRevokeSession();
  const revokeOthers = useRevokeOtherSessions();

  const list = sessions.data ?? [];
  const others = list.filter((s) => !s.current);
  const error = sessions.error ?? revoke.error ?? revokeOthers.error;

  const onRevoke = async (s: Session) => {
    if (await confirm({ title: `Sign out ${s.deviceName}?`, confirmLabel: 'Sign out', destructive: true })) {
      revoke.mutate(s.id);
    }
  };
  const onRevokeOthers = async () => {
    const ok = await confirm({
      title: 'Sign out of all other devices?',
      message: 'You stay signed in on this device.',
      confirmLabel: 'Sign out others',
      destructive: true,
    });
    if (ok) revokeOthers.mutate();
  };

  if (sessions.isPending) return <SkeletonList title="Devices" rows={2} sublabel />;
  if (sessions.data === undefined) {
    return (
      <ListSection title="Devices" footer={errorMessage(sessions.error)}>
        <ListRow label="Try again" action="default" onPress={() => sessions.refetch()} disabled={sessions.isFetching} />
      </ListSection>
    );
  }

  return (
    <ListSection
      title="Devices"
      footer={error ? errorMessage(error) : 'Devices stay signed in for 90 days after they were last used.'}
    >
      {list.map((s) => (
        <ListRow
          key={s.id}
          label={s.deviceName}
          sublabel={s.current ? 'This device' : lastActive(s.lastUsedAt)}
          detail={
            s.current ? null : (
              <Button
                variant="plain"
                destructive
                title="Sign out"
                onPress={() => onRevoke(s)}
                disabled={revoke.isPending && revoke.variables === s.id}
              />
            )
          }
        />
      ))}
      {others.length > 0 ? (
        <ListRow
          label="Sign out of all other devices"
          action="destructive"
          onPress={onRevokeOthers}
          disabled={revokeOthers.isPending}
        />
      ) : null}
    </ListSection>
  );
}

const useStyles = createStyles((c) => ({
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: space.lg,
    marginTop: space.lg,
  },
  pressed: {
    opacity: 0.7,
  },
  avatarBusy: {
    position: 'absolute',
    top: 0,
    left: 0,
    width: AVATAR_SIZE,
    height: AVATAR_SIZE,
    borderRadius: radius.pill,
    backgroundColor: 'rgba(0, 0, 0, 0.35)',
    alignItems: 'center',
    justifyContent: 'center',
  },
  headerText: {
    flex: 1,
    alignItems: 'flex-start',
  },
  photoActions: {
    flexDirection: 'row',
    // Line the plain buttons' text up with the title above.
    marginLeft: -space.md,
  },
  avatarError: {
    marginTop: space.sm,
  },
  fill: {
    flex: 1,
  },
  status: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: space.sm,
  },
  statusDot: {
    width: 8,
    height: 8,
    borderRadius: radius.pill,
    backgroundColor: c.textTertiary,
  },
}));
