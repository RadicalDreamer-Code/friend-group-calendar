import type { ErrorBoundaryProps } from 'expo-router';
import { router } from 'expo-router';
import { View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';

import { createStyles } from '@/theme/ThemeProvider';
import { space } from '@/theme/tokens';

import { Button } from './Button';
import { EmptyState } from './EmptyState';

/**
 * Shown instead of a screen that crashed while rendering, so a bug shows a way
 * out rather than a blank page. Routes export it as their `ErrorBoundary`.
 */
export function ScreenError({ error, retry }: ErrorBoundaryProps) {
  const styles = useStyles();
  const insets = useSafeAreaInsets();
  if (__DEV__) console.error(error);
  return (
    <View style={[styles.root, { paddingTop: insets.top, paddingBottom: insets.bottom }]}>
      <EmptyState
        tone="danger"
        icon="bug-outline"
        title="This screen ran into a problem"
        message="Try again, or go back to your calendars. If it keeps happening, reloading the app usually helps."
        action={
          <View style={styles.actions}>
            <Button title="Try again" onPress={retry} />
            <Button
              variant="plain"
              title="Go to calendars"
              onPress={() => {
                // Back down the calendars stack, or over to its tab from Friends or Profile.
                if (router.canDismiss()) router.dismissTo('/');
                else router.navigate('/');
                // The boundary stays up until told to render the (now different) screen again.
                retry();
              }}
            />
          </View>
        }
      />
    </View>
  );
}

const useStyles = createStyles((c) => ({
  root: {
    flex: 1,
    justifyContent: 'center',
    backgroundColor: c.background,
  },
  actions: {
    alignItems: 'center',
    gap: space.sm,
  },
}));
