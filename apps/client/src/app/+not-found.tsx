import { router } from 'expo-router';
import { View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';

import { Button } from '@/components/Button';
import { EmptyState } from '@/components/EmptyState';
import { createStyles } from '@/theme/ThemeProvider';

/** Any address that matches no screen, e.g. a mistyped or outdated link. */
export default function NotFoundScreen() {
  const styles = useStyles();
  const insets = useSafeAreaInsets();
  return (
    <View style={[styles.root, { paddingTop: insets.top, paddingBottom: insets.bottom }]}>
      <EmptyState
        icon="compass-outline"
        title="Page not found"
        message="This link doesn’t lead anywhere. It may be mistyped, or the page has moved."
        action={<Button title="Go to the app" onPress={() => router.replace('/')} />}
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
}));
