import { router } from 'expo-router';
import { View } from 'react-native';

import { useUser } from '@/auth/session';
import { Button } from '@/components/Button';
import { Page } from '@/components/Page';
import { Text } from '@/components/Text';
import { UsernameForm } from '@/components/UsernameForm';
import { createStyles } from '@/theme/ThemeProvider';
import { space } from '@/theme/tokens';

export default function EditUsernameScreen() {
  const styles = useStyles();
  const user = useUser();
  const close = () => (router.canGoBack() ? router.back() : router.replace('/profile'));

  return (
    <Page title="Username">
      <Text variant="subhead" tone="secondary" style={styles.intro}>
        Friends find and invite you by your username.
      </Text>
      <UsernameForm initial={user.username ?? ''} submitLabel="Save" onSaved={close} />
      <View style={styles.cancel}>
        <Button variant="plain" title="Cancel" onPress={close} />
      </View>
    </Page>
  );
}

const useStyles = createStyles(() => ({
  intro: {
    marginBottom: space.xl,
  },
  cancel: {
    marginTop: space.sm,
    alignItems: 'center',
  },
}));
