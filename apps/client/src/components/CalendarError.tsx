import { router } from 'expo-router';
import { View } from 'react-native';

import { isCalendarGone } from '@/api/calendars';
import { errorMessage } from '@/api/client';
import { createStyles } from '@/theme/ThemeProvider';
import { space } from '@/theme/tokens';

import { Button } from './Button';
import { EmptyState } from './EmptyState';
import { Text } from './Text';

/** For calendar screens whose calendar could not be loaded. */
export function CalendarError({ error, onRetry }: { error: unknown; onRetry: () => void }) {
  const styles = useStyles();
  if (isCalendarGone(error)) {
    return (
      <EmptyState
        icon="calendar-clear-outline"
        title="Calendar not available"
        message="It was deleted, or you are no longer a member."
        action={<Button variant="secondary" title="Back to calendars" onPress={() => router.dismissTo('/')} />}
      />
    );
  }
  return (
    <View style={styles.root}>
      <Text tone="secondary" style={styles.message}>
        {errorMessage(error)}
      </Text>
      <Button variant="secondary" title="Try again" onPress={onRetry} />
    </View>
  );
}

const useStyles = createStyles(() => ({
  root: {
    alignItems: 'center',
    gap: space.lg,
    paddingVertical: space.xxxl,
  },
  message: {
    textAlign: 'center',
  },
}));
