import { router } from 'expo-router';
import { useState } from 'react';
import { View } from 'react-native';

import { calendarNameMaxLength, useCreateCalendar } from '@/api/calendars';
import { errorMessage } from '@/api/client';
import { Button } from '@/components/Button';
import { Page } from '@/components/Page';
import { Text } from '@/components/Text';
import { TextField } from '@/components/TextField';
import { haptics } from '@/lib/haptics';
import { useEscapeKey } from '@/lib/useEscapeKey';
import { createStyles } from '@/theme/ThemeProvider';
import { space } from '@/theme/tokens';

export { ScreenError as ErrorBoundary } from '@/components/ScreenError';

export default function NewCalendarScreen() {
  const styles = useStyles();
  const [name, setName] = useState('');
  const create = useCreateCalendar();
  const trimmed = name.trim();
  const close = () => (router.canGoBack() ? router.back() : router.replace('/'));
  useEscapeKey(close);

  const submit = () => {
    if (trimmed === '' || create.isPending) return;
    create.mutate(trimmed, {
      onSuccess: (calendar) => {
        haptics.success();
        // Next stop: inviting people, from the new calendar's settings,
        // which lead back to its month view.
        router.dismissTo('/');
        router.push({ pathname: '/calendars/[id]', params: { id: calendar.id } });
        router.push({ pathname: '/calendars/[id]/settings', params: { id: calendar.id } });
      },
    });
  };

  return (
    <Page title="New calendar" sheet>
      <Text variant="subhead" tone="secondary" style={styles.intro}>
        You can invite friends once it is created.
      </Text>
      <View style={styles.form}>
        <TextField
          label="Name"
          value={name}
          onChangeText={(text) => {
            setName(text);
            if (create.isError) create.reset();
          }}
          onSubmitEditing={submit}
          placeholder="e.g. Ski trip, Board game nights"
          autoFocus
          autoCapitalize="sentences"
          maxLength={calendarNameMaxLength}
          returnKeyType="done"
          error={create.error ? errorMessage(create.error) : null}
        />
        <Button title="Create" onPress={submit} loading={create.isPending} disabled={trimmed === ''} />
      </View>
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
  form: {
    gap: space.lg,
  },
  cancel: {
    marginTop: space.sm,
    alignItems: 'center',
  },
}));
