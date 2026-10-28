import { router } from 'expo-router';

import { errorMessage, type ApiToken } from '@/api/client';
import { maxTokens, useApiTokens, useRevokeApiToken } from '@/api/tokens';
import { Button } from '@/components/Button';
import { EmptyState } from '@/components/EmptyState';
import { ErrorState, RefreshError } from '@/components/ErrorState';
import { IconButton } from '@/components/IconButton';
import { ListRow, ListSection } from '@/components/ListSection';
import { Page } from '@/components/Page';
import { SkeletonList } from '@/components/Skeleton';
import { confirm } from '@/lib/confirm';
import { shortDate, timeAgo } from '@/lib/format';
import { backTo } from '@/lib/navigation';

export { ScreenError as ErrorBoundary } from '@/components/ScreenError';

const TITLE = 'AI assistants';

export default function AccessTokensScreen() {
  const tokens = useApiTokens();
  const back = { label: 'Profile', onPress: () => backTo('/profile') };
  const create = () => router.push('/new-token');
  const list = tokens.data ?? [];
  const full = list.length >= maxTokens;

  return (
    <Page
      title={TITLE}
      back={back}
      action={list.length > 0 && !full ? <IconButton icon="add" label="New token" accent onPress={create} /> : null}
    >
      {tokens.isPending ? (
        <SkeletonList rows={2} sublabel />
      ) : tokens.data === undefined ? (
        <ErrorState error={tokens.error} onRetry={tokens.refetch} retrying={tokens.isFetching} />
      ) : (
        <>
          {tokens.isRefetchError ? (
            <RefreshError error={tokens.error} onRetry={tokens.refetch} retrying={tokens.isFetching} />
          ) : null}
          {list.length === 0 ? (
            <EmptyState
              icon="sparkles-outline"
              title="Connect an AI assistant"
              message="A token lets Claude or another AI assistant see your calendars and their entries, and add entries where you’re an owner or editor. You can revoke it at any time."
              action={<Button title="New token" onPress={create} />}
            />
          ) : (
            <TokenList tokens={list} full={full} />
          )}
        </>
      )}
    </Page>
  );
}

function TokenList({ tokens, full }: { tokens: ApiToken[]; full: boolean }) {
  const revoke = useRevokeApiToken();

  const onRevoke = async (t: ApiToken) => {
    const ok = await confirm({
      title: `Revoke ${t.name}?`,
      message: 'Assistants that use this token lose access right away.',
      confirmLabel: 'Revoke',
      destructive: true,
    });
    if (ok) revoke.mutate(t.id);
  };

  const footer = revoke.error
    ? errorMessage(revoke.error)
    : full
      ? `You have ${maxTokens} tokens, the most there can be. Revoke one to make a new one.`
      : 'A token can see your calendars and their entries, and add entries where you’re an owner or editor.';

  return (
    <ListSection title="Tokens" footer={footer}>
      {tokens.map((t) => (
        <ListRow
          key={t.id}
          label={t.name}
          sublabel={[
            `Created ${shortDate(t.createdAt)}`,
            t.lastUsedAt ? `used ${timeAgo(t.lastUsedAt)}` : 'never used',
            `expires ${shortDate(t.expiresAt)}`,
          ].join(' · ')}
          detail={
            <Button
              variant="plain"
              destructive
              title="Revoke"
              aria-label={`Revoke ${t.name}`}
              onPress={() => onRevoke(t)}
              disabled={revoke.isPending && revoke.variables === t.id}
            />
          }
        />
      ))}
    </ListSection>
  );
}
