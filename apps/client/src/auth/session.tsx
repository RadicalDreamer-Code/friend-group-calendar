import { useQuery, useQueryClient } from '@tanstack/react-query';
import { createContext, use, useEffect, useMemo, useState, type ReactNode } from 'react';

import { api, ApiError, setUnauthorizedHandler, unwrap, type User } from '@/api/client';

import { clearToken, loadToken } from './token';

/**
 * - loading: checking for an existing session
 * - signedOut: show the sign-in screens
 * - needsUsername: signed in for the first time; pick a username before anything else
 * - signedIn: the app
 * - unreachable: the session could not be checked, e.g. offline at startup; nobody is signed out for that
 */
export type SessionStatus = 'loading' | 'signedOut' | 'needsUsername' | 'signedIn' | 'unreachable';

export const meQueryKey = ['me'] as const;

type SessionContextValue = {
  status: SessionStatus;
  /** The signed-in user; null while loading or signed out. */
  user: User | null;
  /** Why the session could not be checked, when `status` is unreachable. */
  error: unknown;
  /** Checks the session again. */
  retry: () => void;
  retrying: boolean;
};

const SessionContext = createContext<SessionContextValue | null>(null);

async function fetchMe(): Promise<User | null> {
  try {
    return await unwrap(api.GET('/me'));
  } catch (e) {
    if (e instanceof ApiError && e.status === 401) return null;
    throw e;
  }
}

export function SessionProvider({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient();
  // Native reads the token from secure storage first; web has nothing to load.
  const [tokenLoaded, setTokenLoaded] = useState(false);
  useEffect(() => {
    loadToken().then(() => setTokenLoaded(true));
  }, []);

  const me = useQuery({
    queryKey: meQueryKey,
    queryFn: fetchMe,
    enabled: tokenLoaded,
    staleTime: 5 * 60_000,
    // A network error must not sign anyone out. Retry briefly, then offer to try again.
    retry: (failures, error) => !(error instanceof ApiError && error.status === 401) && failures < 2,
  });

  // A session revoked elsewhere (or expired) turns any request into a 401.
  useEffect(() => {
    setUnauthorizedHandler(() => {
      clearToken();
      queryClient.setQueryData(meQueryKey, null);
    });
    return () => setUnauthorizedHandler(null);
  }, [queryClient]);

  const user = me.data ?? null;
  // Keep showing the last known state while offline instead of bouncing to sign-in.
  // 401s resolve to null above, so an error here means the API could not be asked.
  const status: SessionStatus =
    me.data === undefined
      ? me.isError
        ? 'unreachable'
        : 'loading'
      : user === null
        ? 'signedOut'
        : user.username === null
          ? 'needsUsername'
          : 'signedIn';

  const { error, refetch, isFetching: retrying } = me;
  const value = useMemo(
    () => ({ status, user, error, retry: () => void refetch(), retrying }),
    [status, user, error, refetch, retrying],
  );
  return <SessionContext value={value}>{children}</SessionContext>;
}

export function useSession(): SessionContextValue {
  const ctx = use(SessionContext);
  if (!ctx) throw new Error('useSession must be used inside <SessionProvider>');
  return ctx;
}

/** The signed-in user, for screens that are only reachable when signed in. */
export function useUser(): User {
  const { user } = useSession();
  if (!user) throw new Error('useUser needs a signed-in user');
  return user;
}
