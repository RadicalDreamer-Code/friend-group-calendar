import { useState } from 'react';
import { View } from 'react-native';

import { useUpdateUsername, useUsernameCheck, type UsernameCheck } from '@/api/account';
import { errorMessage } from '@/api/client';
import { createStyles } from '@/theme/ThemeProvider';
import { space } from '@/theme/tokens';

import { Button } from './Button';
import { TextField } from './TextField';

const hints: Record<UsernameCheck, string> = {
  idle: '3–20 characters: letters, numbers, _ and .',
  invalid: 'Use 3–20 characters: letters, numbers, _ and .',
  checking: 'Checking…',
  available: 'Available',
  taken: 'That username is taken.',
  unknown: '3–20 characters: letters, numbers, _ and .',
};

type UsernameFormProps = {
  initial?: string;
  submitLabel: string;
  onSaved?: () => void;
};

/** Username field with a live availability check, shared by onboarding and the profile. */
export function UsernameForm({ initial = '', submitLabel, onSaved }: UsernameFormProps) {
  const styles = useStyles();
  const [username, setUsername] = useState(initial);
  const check = useUsernameCheck(username);
  const save = useUpdateUsername();
  const unchanged = initial !== '' && username === initial;

  const submit = () => {
    if (check === 'invalid' || check === 'taken' || save.isPending || unchanged) return;
    save.mutate(username, { onSuccess: () => onSaved?.() });
  };

  const fieldError = save.error
    ? errorMessage(save.error)
    : check === 'taken' || (check === 'invalid' && username.length >= 3)
      ? hints[check]
      : null;

  return (
    <View style={styles.root}>
      <TextField
        label="Username"
        value={username}
        onChangeText={(text) => {
          setUsername(text.trim().toLowerCase());
          if (save.isError) save.reset();
        }}
        onSubmitEditing={submit}
        placeholder="e.g. ada.l"
        autoFocus
        autoCapitalize="none"
        autoCorrect={false}
        autoComplete="username-new"
        textContentType="username"
        maxLength={20}
        returnKeyType="done"
        hint={unchanged ? 'This is your current username.' : hints[check]}
        hintSuccess={check === 'available' && !unchanged}
        error={fieldError}
      />
      <Button
        title={submitLabel}
        onPress={submit}
        loading={save.isPending}
        disabled={unchanged || check === 'idle' || check === 'invalid' || check === 'taken' || check === 'checking'}
      />
    </View>
  );
}

const useStyles = createStyles(() => ({
  root: {
    gap: space.lg,
  },
}));
