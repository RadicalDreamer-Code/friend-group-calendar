import Constants from 'expo-constants';
import { View } from 'react-native';

import { useServerStatus, type ServerStatus } from '@/api/health';
import { Icon } from '@/components/Icon';
import { ListRow, ListSection } from '@/components/ListSection';
import { Page } from '@/components/Page';
import { SegmentedControl } from '@/components/SegmentedControl';
import { Text } from '@/components/Text';
import { createStyles, useTheme, type ThemePreference } from '@/theme/ThemeProvider';
import { radius, space } from '@/theme/tokens';

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

export default function ProfileScreen() {
  const styles = useStyles();
  const { colors, preference, setPreference } = useTheme();
  const status = useServerStatus();

  const statusColor = {
    checking: colors.textTertiary,
    connected: colors.success,
    degraded: colors.danger,
    unreachable: colors.danger,
  }[status];

  return (
    <Page title="Profile">
      <View style={styles.header}>
        <View style={styles.avatar}>
          <Icon name="person" size={34} color={colors.textTertiary} />
        </View>
        <View style={styles.headerText}>
          <Text variant="title">Guest</Text>
          <Text variant="subhead" tone="secondary">
            Sign in with your email to get started.
          </Text>
        </View>
      </View>

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
    </Page>
  );
}

const useStyles = createStyles((c) => ({
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: space.lg,
    marginTop: space.lg,
  },
  avatar: {
    width: 64,
    height: 64,
    borderRadius: radius.pill,
    backgroundColor: c.fill,
    alignItems: 'center',
    justifyContent: 'center',
  },
  headerText: {
    flex: 1,
    gap: space.xxs,
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
  },
}));
