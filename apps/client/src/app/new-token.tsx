import { router } from 'expo-router';
import { useState } from 'react';
import { View } from 'react-native';

import { errorMessage } from '@/api/client';
import { claudeCodeCommand, mcpUrl, tokenNameMaxLength, useCreateApiToken, type TokenExpiry } from '@/api/tokens';
import { Button } from '@/components/Button';
import { CopyField } from '@/components/CopyField';
import { ListRow, ListSection } from '@/components/ListSection';
import { Page } from '@/components/Page';
import { SegmentedControl } from '@/components/SegmentedControl';
import { Text } from '@/components/Text';
import { TextField } from '@/components/TextField';
import { haptics } from '@/lib/haptics';
import { useEscapeKey } from '@/lib/useEscapeKey';
import { createStyles } from '@/theme/ThemeProvider';
import { space } from '@/theme/tokens';

export { ScreenError as ErrorBoundary } from '@/components/ScreenError';

const expiryOptions = [
  { value: '30', label: '30 days' },
  { value: '90', label: '90 days' },
  { value: '365', label: '1 year' },
] as const;

type ExpiryValue = (typeof expiryOptions)[number]['value'];

export default function NewTokenScreen() {
  const close = () => (router.canGoBack() ? router.back() : router.replace('/profile/access-tokens'));
  useEscapeKey(close);
  const [token, setToken] = useState<string | null>(null);

  return (
    <Page title={token ? 'Token created' : 'New token'} sheet>
      {token ? <Created token={token} onDone={close} /> : <Form onCreated={setToken} onCancel={close} />}
    </Page>
  );
}

function Form({ onCreated, onCancel }: { onCreated: (token: string) => void; onCancel: () => void }) {
  const styles = useStyles();
  const [name, setName] = useState('');
  const [expiry, setExpiry] = useState<ExpiryValue>('90');
  const create = useCreateApiToken();
  const trimmed = name.trim();

  const submit = () => {
    if (trimmed === '' || create.isPending) return;
    create.mutate(
      { name: trimmed, expiresInDays: Number(expiry) as TokenExpiry },
      {
        onSuccess: (created) => {
          haptics.success();
          onCreated(created.token);
        },
      },
    );
  };

  return (
    <>
      <Text variant="subhead" tone="secondary" style={styles.intro}>
        An AI assistant like Claude can use a token to see your calendars and their entries, and to add entries where
        you’re an owner or editor.
      </Text>
      <View style={styles.form}>
        <TextField
          label="Name"
          hint="So you can tell your tokens apart."
          value={name}
          onChangeText={(text) => {
            setName(text);
            if (create.isError) create.reset();
          }}
          onSubmitEditing={submit}
          placeholder="e.g. Claude Code on my laptop"
          autoFocus
          autoCapitalize="sentences"
          maxLength={tokenNameMaxLength}
          returnKeyType="done"
          error={create.error ? errorMessage(create.error) : null}
        />
        <ListSection title="Expires after">
          <ListRow>
            <View style={styles.fill}>
              <SegmentedControl
                options={expiryOptions}
                value={expiry}
                onChange={setExpiry}
                accessibilityLabel="Expires after"
              />
            </View>
          </ListRow>
        </ListSection>
        <Button title="Create token" onPress={submit} loading={create.isPending} disabled={trimmed === ''} />
      </View>
      <View style={styles.cancel}>
        <Button variant="plain" title="Cancel" onPress={onCancel} />
      </View>
    </>
  );
}

function Created({ token, onDone }: { token: string; onDone: () => void }) {
  const styles = useStyles();
  return (
    <>
      <Text variant="subhead" tone="secondary" style={styles.intro} role="alert">
        Copy your token now. It won’t be shown again.
      </Text>
      <View style={styles.sections}>
        <CopyField label="Token" value={token} />

        <View style={styles.section}>
          <Text variant="headline" role="heading" aria-level={2}>
            Claude Code
          </Text>
          <Text variant="subhead" tone="secondary">
            Run this in a terminal to add the calendar to Claude Code.
          </Text>
          <CopyField label="Command" value={claudeCodeCommand(token)} />
        </View>

        <View style={styles.section}>
          <Text variant="headline" role="heading" aria-level={2}>
            Any other MCP client
          </Text>
          <Text variant="subhead" tone="secondary">
            Add a remote server (Streamable HTTP) with this address and header.
          </Text>
          <CopyField label="Server URL" value={mcpUrl} />
          <CopyField label="Header" value={`Authorization: Bearer ${token}`} />
        </View>

        <Button title="Done" onPress={onDone} />
      </View>
    </>
  );
}

const useStyles = createStyles(() => ({
  intro: {
    marginBottom: space.xl,
  },
  form: {
    gap: space.lg,
  },
  fill: {
    flex: 1,
  },
  sections: {
    gap: space.xl,
  },
  section: {
    gap: space.md,
  },
  cancel: {
    marginTop: space.sm,
    alignItems: 'center',
  },
}));
