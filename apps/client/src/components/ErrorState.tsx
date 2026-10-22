import { View } from 'react-native';

import { errorMessage, isNetworkError } from '@/api/client';
import { createStyles, useTheme } from '@/theme/ThemeProvider';
import { hairline, radius, space } from '@/theme/tokens';

import { Button } from './Button';
import { EmptyState } from './EmptyState';
import { Icon } from './Icon';
import { Text } from './Text';

type ErrorStateProps = {
  error: unknown;
  onRetry: () => void;
  /** Shows the retry button as busy while the refetch runs. */
  retrying?: boolean;
  /** Less padding, for a part of a screen, like the selected day's entries. */
  compact?: boolean;
};

/** In place of content that could not be loaded at all: what went wrong, and a way to try again. */
export function ErrorState({ error, onRetry, retrying, compact }: ErrorStateProps) {
  const offline = isNetworkError(error);
  return (
    <EmptyState
      tone="danger"
      compact={compact}
      icon={offline ? 'cloud-offline-outline' : 'alert-circle-outline'}
      title={offline ? 'Can’t reach the server' : 'Something went wrong'}
      message={offline ? 'Check your internet connection, then try again.' : errorMessage(error)}
      action={<Button variant="secondary" title="Try again" onPress={() => onRetry()} loading={retrying} />}
    />
  );
}

/**
 * Above content that is still shown from before, when refreshing it failed,
 * e.g. while offline. Keeps the screen usable instead of replacing it.
 */
export function RefreshError({
  error,
  onRetry,
  retrying,
}: {
  error: unknown;
  onRetry: () => void;
  retrying?: boolean;
}) {
  const styles = useStyles();
  const { colors } = useTheme();
  const offline = isNetworkError(error);
  return (
    <View style={styles.banner} role="alert">
      <Icon name={offline ? 'cloud-offline-outline' : 'alert-circle-outline'} size={20} color={colors.danger} />
      <Text variant="footnote" tone="secondary" style={styles.bannerText}>
        {offline ? 'You’re offline or the server is down. ' : 'Couldn’t refresh. '}
        Showing what was loaded before.
      </Text>
      <Button variant="plain" size="small" title="Retry" onPress={() => onRetry()} loading={retrying} />
    </View>
  );
}

const useStyles = createStyles((c) => ({
  banner: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: space.md,
    marginTop: space.lg,
    paddingLeft: space.lg,
    paddingRight: space.xs,
    paddingVertical: space.sm,
    borderRadius: radius.md,
    backgroundColor: c.surface,
    borderWidth: hairline,
    borderColor: c.hairline,
  },
  bannerText: {
    flex: 1,
  },
}));
